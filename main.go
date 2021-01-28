/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	currentruntime "runtime"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	jenkinsv1alpha2 "github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/controllers"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/constants"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/event"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications"
	e "github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"
	"github.com/jenkinsci/jenkins-automation-operator/version"

	// sdkVersion "github.com/operator-framework/operator-sdk/version"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jenkinsv1alpha2.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	parseFlags(metricsAddr, enableLeaderElection)
	debug := pflag.Bool("debug", false, "Set log level to debug")

	setupLog.Info("Registering Components.")
	manager := initManager(metricsAddr, enableLeaderElection)
	client := manager.GetClient()
	restClient := GetRestClient(debug)
	eventsRecorder := getEventsRecorder(restClient, debug)
	checkAvailableFeatures(manager)
	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get config"), *debug)
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create Kubernetes client set"), *debug)
	}
	checkRouteAPIAvailable(clientSet)
	checkPrometheusAPIAvailable(clientSet)
	notificationsChannel := make(chan e.Event)
	go notifications.Listen(notificationsChannel, eventsRecorder, client)

	// setup Jenkins controller
	setupJenkinsRenconciler(manager, notificationsChannel)
	setupJenkinsImageRenconciler(manager)
	setupJenkinsBackupRenconciler(manager)
	setupJenkinsRestoreRenconciler(manager)
	// start the Cmd
	setupLog.Info("Starting the Cmd.")
	runMananger(manager)
	// +kubebuilder:scaffold:builder
}

func checkAvailableFeatures(manager manager.Manager) {
	if resources.IsImageRegistryAvailable(manager) {
		setupLog.Info("Internal Image Registry found: It is very likely that we are running on OpenShift")
		setupLog.Info("If JenkinsImages are built without specified destination, they will be pushed into it.")
	}
}

func checkRouteAPIAvailable(clientSet *kubernetes.Clientset) {
	if resources.IsRouteAPIAvailable(clientSet) {
		setupLog.Info("Route API found: Route creation will be performed")
	}
}

func checkPrometheusAPIAvailable(clientSet *kubernetes.Clientset) {
	if base.IsPrometheusAPIAvailable(clientSet) {
		setupLog.Info("Prometheus API found: ServiceMonitor creation will be performed")
	}
}

func getEventsRecorder(cfg *rest.Config, debug *bool) event.Recorder {
	events, err := event.New(cfg, constants.OperatorName)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"), *debug)
	}

	return events
}

func GetRestClient(debug *bool) *rest.Config {
	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get config"), *debug)
	}

	return cfg
}

func parseFlags(metricsAddr string, enableLeaderElection bool) {
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()
	ctrl.SetLogger(kzap.New(kzap.UseDevMode(true)))
}

func initManager(metricsAddr string, enableLeaderElection bool) manager.Manager {
	printInfo()
	mgr, err := startManager(metricsAddr, enableLeaderElection)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	return mgr
}

func runMananger(mgr manager.Manager) {
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	setupLog.Info("manager started")
}

func startManager(metricsAddr string, enableLeaderElection bool) (manager.Manager, error) {
	options := ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "9cf053ac.jenkins.io",
		Namespace:          getWatchNamespace(), // namespaced-scope when the value is not an empty string
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	return mgr, err
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() string {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	ns, _ := os.LookupEnv(watchNamespaceEnvVar)
	return ns
}

func setupJenkinsRenconciler(mgr manager.Manager, channel chan e.Event) {
	if err := newJenkinsReconciler(mgr, channel).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jenkins")
		os.Exit(1)
	}
}

func newJenkinsReconciler(mgr manager.Manager, channel chan e.Event) *controllers.JenkinsReconciler {
	return &controllers.JenkinsReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("Jenkins"),
		Scheme:             mgr.GetScheme(),
		NotificationEvents: channel,
	}
}

func setupJenkinsImageRenconciler(mgr manager.Manager) {
	if err := newJenkinsImageRenconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jenkins")
		os.Exit(1)
	}
}

func newJenkinsImageRenconciler(mgr manager.Manager) *controllers.JenkinsImageReconciler {
	return &controllers.JenkinsImageReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("JenkinsImage"),
		Scheme: mgr.GetScheme(),
	}
}

func setupJenkinsBackupRenconciler(mgr manager.Manager) {
	if err := newJenkinsBackupRenconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Backup")
		os.Exit(1)
	}
}

func newJenkinsBackupRenconciler(mgr manager.Manager) *controllers.BackupReconciler {
	return &controllers.BackupReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Backup"),
		Scheme: mgr.GetScheme(),
	}
}

func setupJenkinsRestoreRenconciler(mgr manager.Manager) {
	if err := newJenkinsRestoreRenconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Restore")
		os.Exit(1)
	}
}

func newJenkinsRestoreRenconciler(mgr manager.Manager) *controllers.RestoreReconciler {
	return &controllers.RestoreReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Restore"),
		Scheme: mgr.GetScheme(),
	}
}

func fatal(err error, debug bool) {
	if debug {
		setupLog.Error(nil, fmt.Sprintf("%+v", err))
	} else {
		setupLog.Error(nil, fmt.Sprintf("%s", err))
	}
	os.Exit(-1)
}

func printInfo() {
	version.Version = "0.7.1"
	setupLog.Info(fmt.Sprintf("Version: %s", version.Version))
	file, _ := filepath.Abs(os.Args[0])
	setupLog.Info(fmt.Sprintf("MD5 checkcsum: %s", md5sum(file)))
	setupLog.Info(fmt.Sprintf("Go Version: %s", currentruntime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", currentruntime.GOOS, currentruntime.GOARCH))
}

func md5sum(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	result := hex.EncodeToString(hash.Sum(nil))
	return result
}
