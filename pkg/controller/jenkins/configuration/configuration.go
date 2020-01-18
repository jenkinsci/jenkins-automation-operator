package configuration

import (
	"bytes"
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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
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
	Config        *rest.Config
}

// RestartJenkinsMasterPod terminate Jenkins master pod and notifies about it
func (c *Configuration) RestartJenkinsMasterPod(reason reason.Reason) error {
	currentJenkinsMasterPod, err := c.getJenkinsMasterPod()
	if err != nil {
		return err
	}

	if c.IsJenkinsTerminating(*currentJenkinsMasterPod) {
		return nil
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

// IsJenkinsTerminating returns true if the Jenkins pod is terminating
func (c *Configuration) IsJenkinsTerminating(pod corev1.Pod) bool {
	return pod.ObjectMeta.DeletionTimestamp != nil
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

// Exec executes command in the given pod and it's container
func (c *Configuration) Exec(podName, containerName string, command []string) (stdout, stderr bytes.Buffer, err error) {
	req := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(c.Jenkins.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.Config, "POST", req.URL())
	if err != nil {
		return stdout, stderr, stackerr.Wrap(err, "pod exec error while creating Executor")
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout, stderr, stackerr.Wrapf(err, "pod exec error operation on stream: stdout '%s' stderr '%s'", stdout.String(), stderr.String())
	}

	return
}

// GetJenkinsMasterContainer returns the Jenkins master container from the CR
func (c *Configuration) GetJenkinsMasterContainer() *v1alpha2.Container {
	if len(c.Jenkins.Spec.Master.Containers) > 0 {
		// the first container is the Jenkins master, it is forced jenkins_controller.go
		return &c.Jenkins.Spec.Master.Containers[0]
	}
	return nil
}
