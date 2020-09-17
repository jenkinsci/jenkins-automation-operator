package casc

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/user"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	jenkinsv1alpha3 "github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	JenkinsReferenceAnnotation = "jenkins.io/jenkins-reference"
)

var logx = log.Log

func Add(mgr manager.Manager, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) error {
	reconciler := newReconciler(mgr, jenkinsAPIConnectionSettings, clientSet, config, notificationEvents)
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) reconcile.Reconciler {
	return &ReconcileCasc{
		client:                       mgr.GetClient(),
		scheme:                       mgr.GetScheme(),
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		clientSet:                    clientSet,
		config:                       config,
		notificationEvents:           notificationEvents,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("casc-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Casc
	err = c.Watch(&source.Kind{Type: &jenkinsv1alpha3.Casc{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCasc implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCasc{}

// ReconcileCasc reconciles a Casc object
type ReconcileCasc struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                       client.Client
	scheme                       *runtime.Scheme
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	clientSet                    kubernetes.Clientset
	config                       rest.Config
	notificationEvents           *chan event.Event
}

// Reconcile reads that state of the cluster for a Casc object and makes changes based on the state read
// and what is in the Casc.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCasc) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := logx.WithValues("cr", request.Name)
	logger.V(log.VDebug).Info("Reconciling Casc")

	// Fetch the Casc instance
	casc := &jenkinsv1alpha3.Casc{}
	err := r.client.Get(context.TODO(), request.NamespacedName, casc)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if casc.Status.Phase == constants.JenkinsStatusCompleted {
		return reconcile.Result{}, nil // Nothing to see here, move along...
	}

	// fetch the jenkins CR
	jenkinsName, _ := GetAnnotation(JenkinsReferenceAnnotation, casc.ObjectMeta)
	message := fmt.Sprintf("Casc configuration references Jenkins instance annotation: %s, jenkinsName: %s", JenkinsReferenceAnnotation, jenkinsName)
	logger.V(log.VDebug).Info(message)
	jenkins := &v1alpha2.Jenkins{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: jenkinsName, Namespace: casc.Namespace}, jenkins)
	if err != nil {
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// check if jenkins Cr is completed
	if jenkins.Status.Phase != constants.JenkinsStatusCompleted {
		message = fmt.Sprintf("Jenkins CR '%s' has not reached status 'Completed' yet. Requeuing casc configuration", jenkinsName)
		logger.V(log.VDebug).Info(message)
		return reconcile.Result{Requeue: true}, nil
	}
	// setControllerReference
	if err := controllerutil.SetControllerReference(jenkins, casc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Execute the casc
	config := configuration.Configuration{
		Client:                       r.client,
		ClientSet:                    r.clientSet,
		Notifications:                r.notificationEvents,
		Jenkins:                      jenkins,
		Casc:                         casc,
		Scheme:                       r.scheme,
		RestConfig:                   r.config,
		JenkinsAPIConnectionSettings: r.jenkinsAPIConnectionSettings,
	}

	jenkinsClient, err := config.GetJenkinsClient()
	if err != nil {
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	logger.V(log.VDebug).Info("Jenkins API client set")

	// Reconcile user configuration
	userConfiguration := user.New(config, jenkinsClient)
	var messages []string
	messages, err = userConfiguration.Validate(jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(messages) > 0 {
		message := "Validation of user configuration failed, please correct Jenkins CR"
		*r.notificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseUser,
			Level:   v1alpha2.NotificationLevelWarning,
			Reason:  reason.NewUserConfigurationFailed(reason.HumanSource, []string{message}, append([]string{message}, messages...)...),
		}

		logger.V(log.VDebug).Info(message)
		for _, msg := range messages {
			logger.V(log.VDebug).Info(msg)
		}
		return reconcile.Result{}, nil // don't requeue
	}

	result, err := userConfiguration.ReconcileCasc()
	if result.Requeue {
		return result, err
	}
	//Update the status
	casc.Status.LastTransitionTime = metav1.Now()
	casc.Status.Phase = constants.JenkinsStatusCompleted
	err = r.client.Update(context.TODO(), casc)
	if err != nil {
		return reconcile.Result{}, err
	}
	//TODO Add fields for time
	//time := casc.Status.ConfigurationCompleteTime.Sub(casc.Status.ConfigurationStartTime)
	time := casc.Status.LastTransitionTime.Sub(casc.CreationTimestamp.Time)
	message = fmt.Sprintf("Configuration as code phase is complete, took %s", time)
	*r.notificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseUser,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewUserConfigurationComplete(reason.OperatorSource, []string{message}),
	}
	logger.V(log.VDebug).Info(message)
	return reconcile.Result{}, nil
}

// GetAnnotation returns the value of an annoation for a given key and true if the key was found
func GetAnnotation(key string, meta metav1.ObjectMeta) (string, bool) {
	for k, value := range meta.Annotations {
		if k == key {
			return value, true
		}
	}
	return "", false
}
