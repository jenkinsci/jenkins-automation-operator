package configuration

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Configuration holds required for Jenkins configuration
type Configuration struct {
	Client        client.Client
	ClientSet     kubernetes.Clientset
	Notifications *chan event.Event
	Jenkins       *v1alpha2.Jenkins
	Scheme        *runtime.Scheme
}

// RestartJenkinsMasterPod terminate Jenkins master pod and notifies about it
func (c *Configuration) RestartJenkinsMasterPod(reason reason.Reason) error {
	currentJenkinsMasterPod, err := c.getJenkinsMasterPod()
	if err != nil {
		return err
	}

	*c.Notifications <- event.Event{
		Jenkins: *c.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason,
	}

	return stackerr.WithStack(c.Client.Delete(context.TODO(), currentJenkinsMasterPod))
}

func (c *Configuration) getJenkinsMasterPod() (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(*c.Jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := c.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: c.Jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

// CreateResource is creating kubernetes resource and references it to Jenkins CR
func (c *Configuration) CreateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// Set Jenkins instance as the owner and controller
	if err := controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme); err != nil {
		return stackerr.WithStack(err)
	}

	return c.Client.Create(context.TODO(), runtimeObj) // don't wrap error
}

// UpdateResource is updating kubernetes resource and references it to Jenkins CR
func (c *Configuration) UpdateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check error(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	return c.Client.Update(context.TODO(), runtimeObj) // don't wrap error
}

// CreateOrUpdateResource is creating or updating kubernetes resource and references it to Jenkins CR
func (c *Configuration) CreateOrUpdateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check error(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	err := c.Client.Create(context.TODO(), runtimeObj)
	if err != nil && errors.IsAlreadyExists(err) {
		return c.UpdateResource(obj)
	} else if err != nil && !errors.IsAlreadyExists(err) {
		return stackerr.WithStack(err)
	}

	return nil
}
