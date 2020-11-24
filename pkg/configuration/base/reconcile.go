package base

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	reconcile "sigs.k8s.io/controller-runtime"
)

const (
	fetchAllPlugins = 1
)

// JenkinsBaseConfigurationReconciler defines values required for Jenkins base configuration.
type JenkinsBaseConfigurationReconciler struct {
	configuration.Configuration
	logger                       logr.Logger
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
}

// New create structure which takes care of base configuration
func New(config configuration.Configuration, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings) *JenkinsBaseConfigurationReconciler {
	return &JenkinsBaseConfigurationReconciler{
		Configuration:                config,
		logger:                       log.Log.WithName(""),
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
	}
}

// Reconcile takes care of base configuration.
func (r *JenkinsBaseConfigurationReconciler) Reconcile(request reconcile.Request) (reconcile.Result, jenkinsclient.Jenkins, error) {
	jenkinsConfig := resources.NewResourceObjectMeta(r.Configuration.Jenkins)
	// Create Necessary Resources
	err := r.ensureResourcesRequiredForJenkinsDeploymentArePresent(jenkinsConfig)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Kubernetes resources are present")

	result, err := r.ensureJenkinsDeploymentIsPresent(jenkinsConfig)
	if err != nil {
		r.logger.V(log.VDebug).Info(fmt.Sprintf("Error when ensuring if Jenkins Deployment is present %s", err))
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins Deployment is present: Requeue result is: %+v", result.Requeue))
	r.logger.V(log.VDebug).Info("Ensuring that Deployment is ready")
	result, err = r.ensureJenkinsDeploymentIsReady()
	if err != nil {
		r.logger.V(log.VDebug).Info(fmt.Sprintf("Error when ensuring that Deployment is ready %s", err))
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Deployment for jenkins.io { %s } is ready ", r.Jenkins.Name))

	jenkinsPod, err := r.Configuration.GetPodByDeployment()
	if err != nil {
		r.logger.V(log.VDebug).Info(fmt.Sprintf("Error when checking if Deployment has Pod : %s", err.Error()))
		return reconcile.Result{}, nil, err
	}

	if jenkinsPod != nil {
		podName := jenkinsPod.Name
		deletionTimestamp := jenkinsPod.DeletionTimestamp
		if deletionTimestamp != nil {
			return reconcile.Result{Requeue: true}, nil, fmt.Errorf("jenkins pod %s in Terminating state with DeletionTimestamp %s", podName, deletionTimestamp)
		}
		r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins Pod created with name : %s", podName))
	} else {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("Jenkins Pod not created for Deployment : %s", r.Jenkins.Name))
		return reconcile.Result{}, nil, err
	}

	return result, nil, err
}

func (r *JenkinsBaseConfigurationReconciler) ensureResourcesRequiredForJenkinsDeploymentArePresent(metaObject metav1.ObjectMeta) error {
	if err := r.createOperatorCredentialsSecret(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Operator credentials secret is present")

	if err := r.createScriptsConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Scripts config map is present")

	if err := r.createInitConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Init configuration config map is present")

	if err := r.createBasePluginsConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Base plugins config map is present")

	if err := r.createRBAC(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Service account, role and role binding are present")

	if err := r.ensureExtraRBAC(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Extra role bindings are present")

	httpServiceName := resources.GetJenkinsHTTPServiceName(r.Configuration.Jenkins)
	if err := r.createService(metaObject, httpServiceName, r.Configuration.Jenkins.Status.Spec.Service); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins HTTP Service is present")

	if err := r.createService(metaObject, resources.GetJenkinsJNLPServiceName(r.Configuration.Jenkins), r.Configuration.Jenkins.Status.Spec.JNLPService); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins JNLP Service is present")

	if resources.RouteAPIAvailable {
		r.logger.V(log.VDebug).Info("Route API is available. Now ensuring route is present")
		if err := r.createRoute(metaObject, httpServiceName, r.Configuration.Jenkins); err != nil {
			return err
		}
		r.logger.V(log.VDebug).Info("Jenkins Route is present")
	}

	return nil
}

func (r *JenkinsBaseConfigurationReconciler) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
	found := &corev1.Secret{}
	err := r.Configuration.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.Configuration.Jenkins), Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, found)

	if err != nil && apierrors.IsNotFound(err) {
		return stackerr.WithStack(r.CreateResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return stackerr.WithStack(err)
	}

	if found.Data[resources.OperatorCredentialsSecretUserNameKey] != nil &&
		found.Data[resources.OperatorCredentialsSecretPasswordKey] != nil {
		return nil
	}
	return stackerr.WithStack(r.UpdateResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
}

func compareImagePullSecrets(expected, actual []corev1.LocalObjectReference) bool {
	for _, expected := range expected {
		found := false
		for _, actual := range actual {
			if expected.Name == actual.Name {
				found = true

				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func compareMap(expected, actual map[string]string) bool {
	for expectedKey, expectedValue := range expected {
		actualValue, found := actual[expectedKey]
		if !found {
			return false
		}
		if expectedValue != actualValue {
			return false
		}
	}

	return true
}

func compareEnv(expected, actual []corev1.EnvVar) bool {
	var actualEnv []corev1.EnvVar
	for _, env := range actual {
		if env.Name == "KUBERNETES_PORT_443_TCP_ADDR" || env.Name == "KUBERNETES_PORT" ||
			env.Name == "KUBERNETES_PORT_443_TCP" || env.Name == "KUBERNETES_SERVICE_HOST" {
			continue
		}
		actualEnv = append(actualEnv, env)
	}
	return reflect.DeepEqual(expected, actualEnv)
}

// CompareContainerVolumeMounts returns true if two containers volume mounts are the same.
func CompareContainerVolumeMounts(expected corev1.Container, actual corev1.Container) bool {
	var withoutServiceAccount []corev1.VolumeMount
	for _, volumeMount := range actual.VolumeMounts {
		if volumeMount.MountPath != "/var/run/secrets/kubernetes.io/serviceaccount" {
			withoutServiceAccount = append(withoutServiceAccount, volumeMount)
		}
	}

	return reflect.DeepEqual(expected.VolumeMounts, withoutServiceAccount)
}

func (r *JenkinsBaseConfigurationReconciler) FilterEvents(source corev1.EventList, jenkinsMasterPod corev1.Pod) []string {
	events := []string{}
	for _, eventItem := range source.Items {
		if r.Configuration.Jenkins.Status.ProvisionStartTime.UTC().After(eventItem.LastTimestamp.UTC()) {
			continue
		}
		if eventItem.Type == corev1.EventTypeNormal {
			continue
		}
		if !strings.HasPrefix(eventItem.ObjectMeta.Name, jenkinsMasterPod.Name) {
			continue
		}
		events = append(events, fmt.Sprintf("Message: %s Subobject: %s", eventItem.Message, eventItem.InvolvedObject.FieldPath))
	}
	return events
}
