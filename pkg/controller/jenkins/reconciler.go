package jenkins

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileJenkins reconciles a Jenkins object.
type ReconcileJenkins struct {
	client                       client.Client
	scheme                       *runtime.Scheme
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	clientSet                    kubernetes.Clientset
	restConfig                   rest.Config
	notificationEvents           *chan event.Event
}

func (r *ReconcileJenkins) newReconcilierConfiguration(jenkins *v1alpha2.Jenkins) configuration.Configuration {
	config := configuration.Configuration{
		Client:                       r.client,
		ClientSet:                    r.clientSet,
		RestConfig:                   r.restConfig,
		JenkinsAPIConnectionSettings: r.jenkinsAPIConnectionSettings,
		Notifications:                r.notificationEvents,
		Jenkins:                      jenkins,
		Scheme:                       r.scheme,
	}
	return config
}

// NewReconciler returns a newReconcilierConfiguration reconcile.Reconciler.
func NewReconciler(mgr manager.Manager, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) reconcile.Reconciler {
	return &ReconcileJenkins{
		client:                       mgr.GetClient(),
		scheme:                       mgr.GetScheme(),
		clientSet:                    clientSet,
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		restConfig:                   config,
		notificationEvents:           notificationEvents,
	}
}

func (r *ReconcileJenkins) sendNewReconcileLoopFailedNotification(jenkins *v1alpha2.Jenkins, reconcileFailLimit uint64, err error) {
	*r.notificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelWarning,
		Reason: reason.NewReconcileLoopFailed(
			reason.OperatorSource,
			[]string{fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %s", reconcileFailLimit, err)},
		),
	}
}

func (r *ReconcileJenkins) sendNewBaseConfigurationCompleteNotification(jenkins *v1alpha2.Jenkins, message string) {
	*r.notificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewBaseConfigurationComplete(reason.OperatorSource, []string{message}),
	}
}

func (r *ReconcileJenkins) sendNewGroovyScriptExecutionFailedNotification(jenkins *v1alpha2.Jenkins, groovyErr *jenkinsclient.GroovyScriptExecutionFailed) {
	*r.notificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelWarning,
		Reason: reason.NewGroovyScriptExecutionFailed(
			reason.OperatorSource,
			[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name)},
			[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed, logs: %+v", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name, groovyErr.Logs)}...,
		),
	}
}
