package jenkins

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
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
	config                       rest.Config
	notificationEvents           *chan event.Event
}

func (r *ReconcileJenkins) newReconcilierConfiguration(jenkins *v1alpha2.Jenkins) configuration.Configuration {
	config := configuration.Configuration{
		Client:                       r.client,
		ClientSet:                    r.clientSet,
		Notifications:                r.notificationEvents,
		Jenkins:                      jenkins,
		Scheme:                       r.scheme,
		Config:                       &r.config,
		JenkinsAPIConnectionSettings: r.jenkinsAPIConnectionSettings,
	}
	return config
}

// newReconciler returns a newReconcilierConfiguration reconcile.Reconciler.
func newReconciler(mgr manager.Manager, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) reconcile.Reconciler {
	return &ReconcileJenkins{
		client:                       mgr.GetClient(),
		scheme:                       mgr.GetScheme(),
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		clientSet:                    clientSet,
		config:                       config,
		notificationEvents:           notificationEvents,
	}
}
