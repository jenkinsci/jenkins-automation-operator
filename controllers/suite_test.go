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

package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"github.com/jenkinsci/jenkins-automation-operator/pkg/constants"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/event"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications"
	e "github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	jenkinsv1alpha2 "github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	restConfig      *rest.Config
	k8sClient       client.Client
	testEnv         *envtest.Environment
	suiteTestLogger = ctrl.Log.WithName("suite_test.go")
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Controller Suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	err = jenkinsv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = jenkinsv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = jenkinsv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	manager, err := getManager()
	Expect(err).ToNot(HaveOccurred())

	registerJenkinsRestoreController(manager)
	registerJenkinsBackupController(manager)
	registerJenkinsBackupVolumeController(manager)
	registerJenkinsImageController(manager)
	registerJenkinsController(manager, restConfig)

	go func() {
		// defer GinkgoRecover()
		err = manager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			Logf("Error while starting manager: %+v", err)
		}
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient := manager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

func registerJenkinsRestoreController(manager manager.Manager) {
	controller := &RestoreReconciler{
		Client: manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Restore"),
		Scheme: manager.GetScheme(),
	}
	err := controller.SetupWithManager(manager)
	Expect(err).ToNot(HaveOccurred())
}

func registerJenkinsBackupController(manager manager.Manager) {
	controller := &BackupReconciler{
		Client: manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Backup"),
		Scheme: manager.GetScheme(),
	}
	err := controller.SetupWithManager(manager)
	Expect(err).ToNot(HaveOccurred())
}

func registerJenkinsBackupVolumeController(manager manager.Manager) {
	controller := &BackupVolumeReconciler{
		Client: manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("BackupVolume"),
		Scheme: manager.GetScheme(),
	}
	err := controller.SetupWithManager(manager)
	Expect(err).ToNot(HaveOccurred())
}

func registerJenkinsImageController(manager manager.Manager) {
	controller := &JenkinsImageReconciler{
		Client: manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("JenkinsImageController"),
		Scheme: manager.GetScheme(),
	}
	err := controller.SetupWithManager(manager)
	Expect(err).ToNot(HaveOccurred())
}

func registerJenkinsController(manager manager.Manager, c *rest.Config) {
	notificationsChannel := make(chan e.Event)
	eventsRecorder := getEventsRecorder(c)
	client := manager.GetClient()
	go notifications.Listen(notificationsChannel, eventsRecorder, client)
	controller := &JenkinsReconciler{
		Client:             client,
		Log:                ctrl.Log.WithName("controllers").WithName("JenkinsReconciler"),
		Scheme:             manager.GetScheme(),
		NotificationEvents: notificationsChannel,
	}
	err := controller.SetupWithManager(manager)
	Expect(err).ToNot(HaveOccurred())
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

// Logf is only inteded to be used for debugging e2e and especially wait functions.
// instead, use the facilities provided by Context() By() etc...
func Logf(format string, a ...interface{}) {
	fmt.Fprintf(GinkgoWriter, "INFO: "+format+"\n", a...)
}

var MetricsBindAddress = ":8888"

func getManager() (manager.Manager, error) {
	k8sManager, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: MetricsBindAddress,
	})
	return k8sManager, err
}

func getEventsRecorder(cfg *rest.Config) event.Recorder {
	events, err := event.New(cfg, constants.OperatorName)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"))
	}

	return events
}

func fatal(err error) {
	suiteTestLogger.Error(nil, fmt.Sprintf("%+v", err))
	os.Exit(-1)
}
