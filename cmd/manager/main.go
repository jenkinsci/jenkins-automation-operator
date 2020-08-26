package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/jenkinsci/kubernetes-operator/pkg/controller/casc"

	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkinsimage"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications"
	e "github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var logger = log.Log.WithName("cmd")

func printInfo() {
	logger.Info(fmt.Sprintf("Version: %s", version.Version))
	logger.Info(fmt.Sprintf("Git commit: %s", version.GitCommit))
	logger.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	logger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	logger.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	hostname := pflag.String("jenkins-api-hostname", "", "Hostname or IP of Jenkins API. It can be service name, node IP or localhost.")
	port := pflag.Int("jenkins-api-port", 0, "The port on which Jenkins API is running. Note: If you want to use nodePort don't set this setting and --jenkins-api-use-nodeport must be true.")
	useNodePort := pflag.Bool("jenkins-api-use-nodeport", false, "Connect to Jenkins API using the service nodePort instead of service port. If you want to set this as true - don't set --jenkins-api-port.")
	debug := pflag.Bool("debug", false, "Set log level to debug")
	pflag.Parse()

	log.SetupLogger(*debug)
	printInfo()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get watch namespace"), *debug)
	}
	logger.Info(fmt.Sprintf("Watch namespace: %v", namespace))

	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get config"), *debug)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "jenkins-operator-lock")
	if err != nil {
		fatal(errors.Wrap(err, "failed to become leader"), *debug)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"), *debug)
	}

	logger.Info("Registering Components.")

	// setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		fatal(errors.Wrap(err, "failed to setup scheme"), *debug)
	}

	// setup events
	events, err := event.New(cfg, constants.OperatorName)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"), *debug)
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create Kubernetes client set"), *debug)
	}

	if resources.IsRouteAPIAvailable(clientSet) {
		logger.Info("Route API found: Route creation will be performed")
	}

	if resources.IsImageRegistryAvailable(clientSet) {
		logger.Info("Internal Image Registry found: It is very likely that we are running on OpenShift")
		logger.Info("If JenkinsImages are built without specified destination, they will be pushed into it.")
	}

	c := make(chan e.Event)
	go notifications.Listen(c, events, mgr.GetClient())

	// validate jenkins API connection
	jenkinsAPIConnectionSettings := client.JenkinsAPIConnectionSettings{Hostname: *hostname, Port: *port, UseNodePort: *useNodePort}
	if err := jenkinsAPIConnectionSettings.Validate(); err != nil {
		fatal(errors.Wrap(err, "invalid command line parameters"), *debug)
	}

	// setup Jenkins controller
	if err := jenkins.Add(mgr, jenkinsAPIConnectionSettings, *clientSet, *cfg, &c); err != nil {
		fatal(errors.Wrap(err, "failed to setup controllers"), *debug)
	}
	// setup JenkinsImage controller
	if err = jenkinsimage.Add(mgr, *clientSet); err != nil {
		fatal(errors.Wrap(err, "failed to setup controllers"), *debug)
	}

	// setup Casc controller
	if err := casc.Add(mgr, jenkinsAPIConnectionSettings, *clientSet, *cfg, &c); err != nil {
		fatal(errors.Wrap(err, "failed to setup controllers"), *debug)
	}

	if err = serveCRMetrics(cfg); err != nil {
		logger.V(log.VWarn).Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		logger.V(log.VWarn).Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		logger.V(log.VWarn).Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			logger.V(log.VWarn).Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}

	logger.Info("Starting the Cmd.")

	// start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		fatal(errors.Wrap(err, "failed to start cmd"), *debug)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	gvks, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// We perform our custom GKV filtering on top of the one performed
	// by operator-sdk code
	filteredGVK := filterGKVsFromAddToScheme(gvks)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	return kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
}

func filterGKVsFromAddToScheme(gvks []schema.GroupVersionKind) []schema.GroupVersionKind {
	// We use gkvFilters to filter from the existing GKVs defined in the used
	// runtime.Schema for the operator. The reason for that is that
	// kube-metrics tries to list all of the defined Kinds in the schemas
	// that are passed, including Kinds that the operator doesn't use and
	// thus the role used the operator doesn't have them set and we don't want
	// to set as they are not used by the operator.
	// For the fields that the filters have we have defined the value '*' to
	// specify any will be a match (accepted)
	matchAnyValue := "*"
	gvkFilters := []schema.GroupVersionKind{
		// Kubernetes Resources
		{Kind: "PersistentVolumeClaim", Version: matchAnyValue},
		{Kind: "ServiceAccount", Version: matchAnyValue},
		{Kind: "Secret", Version: matchAnyValue},
		{Kind: "Pod", Version: matchAnyValue},
		{Kind: "ConfigMap", Version: matchAnyValue},
		{Kind: "Service", Version: matchAnyValue},
		{Group: "apps", Kind: "Deployment", Version: matchAnyValue},
		// Openshift Resources
		{Group: "route.openshift.io", Kind: "Route", Version: matchAnyValue},
		{Group: "image.openshift.io", Kind: "ImageStream", Version: matchAnyValue},
		// Custom Resources
		{Group: "jenkins.io", Kind: "Jenkins", Version: matchAnyValue},
		{Group: "jenkins.io", Kind: "JenkinsImage", Version: matchAnyValue},
		{Group: "jenkins.io", Kind: "Casc", Version: matchAnyValue},
	}

	ownGVKs := []schema.GroupVersionKind{}
	for _, gvk := range gvks {
		for _, gvkFilter := range gvkFilters {
			match := true
			if gvkFilter.Kind == matchAnyValue && gvkFilter.Group == matchAnyValue && gvkFilter.Version == matchAnyValue {
				logger.V(1).Info("gvkFilter should at least have one of its fields defined. Skipping...")
				match = false
			} else {
				if gvkFilter.Kind != matchAnyValue && gvkFilter.Kind != gvk.Kind {
					match = false
				}
				if gvkFilter.Group != matchAnyValue && gvkFilter.Group != gvk.Group {
					match = false
				}
				if gvkFilter.Version != matchAnyValue && gvkFilter.Version != gvk.Version {
					match = false
				}
			}
			if match {
				ownGVKs = append(ownGVKs, gvk)
			}
		}
	}

	return ownGVKs
}

func fatal(err error, debug bool) {
	if debug {
		logger.Error(nil, fmt.Sprintf("%+v", err))
	} else {
		logger.Error(nil, fmt.Sprintf("%s", err))
	}
	os.Exit(-1)
}
