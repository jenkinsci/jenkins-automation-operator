package configuration

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"

	stackerr "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

// Configuration holds required for Jenkins configuration.
type Configuration struct {
	Client                       client.Client
	ClientSet                    kubernetes.Clientset
	RestConfig                   rest.Config
	JenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	Jenkins                      *v1alpha2.Jenkins
	Casc                         *v1alpha3.Casc
	Scheme                       *runtime.Scheme
	Notifications                *chan event.Event
}

var (
	logx   = log.Log
	logger = logx.WithName("configuration.go")
)

// GetJenkinsMasterPod gets the jenkins master pod.
func (c *Configuration) GetJenkinsMasterPod() (*corev1.Pod, error) {
	pod, err := c.GetPodByDeployment()
	return pod, err
}

// GetJenkinsDeployment gets the jenkins master deployment.
func (c *Configuration) GetJenkinsDeployment() (*appsv1.Deployment, error) {
	logger.V(log.VDebug).Info(fmt.Sprintf("Getting JenkinsDeploymentName for : %+v", c.Jenkins.Name))
	jenkinsDeploymentName := resources.GetJenkinsDeploymentName(c.Jenkins.Name)
	currentJenkinsDeployment := &appsv1.Deployment{}
	err := c.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsDeploymentName, Namespace: c.Jenkins.Namespace}, currentJenkinsDeployment)
	if err != nil {
		return nil, err
	}
	return currentJenkinsDeployment, nil
}

// IsJenkinsTerminating returns true if the Jenkins pod is terminating.
func (c *Configuration) IsJenkinsTerminating(pod *corev1.Pod) bool {
	return pod.ObjectMeta.DeletionTimestamp != nil
}

// CreateResource is creating kubernetes resource and references it to Jenkins CR
func (c *Configuration) CreateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// Set Jenkins instance as the owner and controller.
	if err := controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme); err != nil {
		return stackerr.WithStack(err)
	}

	return c.Client.Create(context.TODO(), runtimeObj) // don't wrap error
}

// UpdateResource is updating kubernetes resource and references it to Jenkins CR.
func (c *Configuration) UpdateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check error(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	return c.Client.Update(context.TODO(), runtimeObj) // don't wrap error
}

// CreateOrUpdateResource is creating or updating kubernetes resource and references it to Jenkins CR.
func (c *Configuration) CreateOrUpdateResource(obj metav1.Object) error {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check error(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	err := c.Client.Create(context.TODO(), runtimeObj)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		return c.UpdateResource(obj)
	} else if err != nil && !k8serrors.IsAlreadyExists(err) {
		return stackerr.WithStack(err)
	}

	return nil
}

// Exec executes command in the given pod and it's container.
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

	exec, err := remotecommand.NewSPDYExecutor(&c.RestConfig, "POST", req.URL())
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

// GetJenkinsMasterContainer returns the Jenkins master container from the CR.
func (c *Configuration) GetJenkinsMasterContainer() *v1alpha2.Container {
	if len(c.Jenkins.Spec.Master.Containers) > 0 {
		// the first container is the Jenkins master, it is forced jenkins_controller.go
		return &c.Jenkins.Spec.Master.Containers[0]
	}
	return nil
}

// GetJenkinsClient gets jenkins client from a configuration.
func (c *Configuration) GetJenkinsClient() (jenkinsclient.Jenkins, error) {
	logger := logx.WithName("")
	strategy := c.Jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy
	logger.V(log.VDebug).Info(fmt.Sprintf("Getting JenkinsClient for : %s with AuthorizationStrategy: %s", c.Jenkins.Name, strategy))
	switch strategy {
	case v1alpha2.ServiceAccountAuthorizationStrategy:
		return c.GetJenkinsClientFromServiceAccount()
	case v1alpha2.CreateUserAuthorizationStrategy:
		return c.GetJenkinsClientFromSecret()
	default:
		errorMessage := fmt.Sprintf("Unrecognized AuthorizationStrategy %s for Jenkins %s: returning nil", c.Jenkins.Name, strategy)
		logger.V(log.VDebug).Info(errorMessage)
		return nil, stackerr.New(errorMessage)
	}
}

func (c *Configuration) getJenkinsAPIUrl() (string, error) {
	var service corev1.Service
	objectKey := types.NamespacedName{
		Namespace: c.Jenkins.ObjectMeta.Namespace,
		Name:      resources.GetJenkinsHTTPServiceName(c.Jenkins),
	}
	err := c.Client.Get(context.TODO(), objectKey, &service)
	if err != nil {
		return "", err
	}
	jenkinsURL := c.JenkinsAPIConnectionSettings.BuildJenkinsAPIUrl(service.Name, service.Namespace, service.Spec.Ports[0].Port, service.Spec.Ports[0].NodePort)
	if prefix, ok := GetJenkinsOpts(*c.Jenkins)["prefix"]; ok {
		jenkinsURL += prefix
	}
	return jenkinsURL, nil
}

// GetJenkinsMasterPodName returns Jenkins pod name for given CR
func (c *Configuration) GetJenkinsMasterPodName() string {
	pod, _ := c.GetPodByDeployment()
	if pod != nil {
		return pod.Name
	}
	return ""
}

func (c *Configuration) GetReplicaSetByDeployment() (appsv1.ReplicaSet, error) {
	deployment, _ := c.GetJenkinsDeployment()
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return appsv1.ReplicaSet{}, err
	}
	replicasSet := appsv1.ReplicaSetList{}
	listOptions := client.ListOptions{LabelSelector: selector}
	err = c.Client.List(context.TODO(), &replicasSet, &listOptions)
	return replicasSet.Items[0], err
}

func (c *Configuration) GetPodByDeployment() (*corev1.Pod, error) {
	replicaSet, _ := c.GetReplicaSetByDeployment()
	selector, _ := metav1.LabelSelectorAsSelector(replicaSet.Spec.Selector)
	listOptions := client.ListOptions{LabelSelector: selector}
	pods := corev1.PodList{}
	pod := corev1.Pod{}
	err := c.Client.List(context.TODO(), &pods, &listOptions)
	if len(pods.Items) == 0 {
		return &pod, stackerr.Errorf("Deployment has no pod attached yet")
	} else if len(pods.Items) > 0 {
		for i, p := range pods.Items {
			logger.V(log.VDebug).Info(fmt.Sprintf("Pod %d with name %s is in Phase %s", i, p.Name, p.Status.Phase))
			if p.DeletionTimestamp != nil {
				return nil, fmt.Errorf("pod %s is in Terminating state with DeletionTimestamp %s, possible Jenkins restart in progress", p.Name, p.DeletionTimestamp)
			}
		}
	}
	return &pod, err
}

// GetJenkinsClientFromServiceAccount gets jenkins client from a serviceAccount.
func (c *Configuration) GetJenkinsClientFromServiceAccount() (jenkinsclient.Jenkins, error) {
	logger.V(log.VDebug).Info("Creating Jenkins Client from serviceAccount")
	jenkinsAPIUrl, err := c.getJenkinsAPIUrl()
	if err != nil {
		return nil, err
	}
	logger.V(log.VDebug).Info(fmt.Sprintf("Creating Jenkins Client from serviceAccount with URL: %+v", jenkinsAPIUrl))
	masterPod, _ := c.GetPodByDeployment()
	podName := masterPod.Name
	token, _, err := c.Exec(podName, resources.JenkinsMasterContainerName, []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"})
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while getJenkinsAPIUrl: %s", err))
		return nil, err
	}
	return jenkinsclient.NewBearerTokenAuthorization(jenkinsAPIUrl, token.String())
}

// GetJenkinsClientFromSecret gets jenkins client from a secret.
func (c *Configuration) GetJenkinsClientFromSecret() (jenkinsclient.Jenkins, error) {
	logger.V(log.VDebug).Info("Creating Jenkins Client from Secret")
	jenkinsAPIUrl, err := c.getJenkinsAPIUrl()
	logger.V(log.VDebug).Info(fmt.Sprintf("Creating Jenkins Client from Secret with URL: %+v", jenkinsAPIUrl))
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while getJenkinsAPIUrl: %+v", err))
		return nil, err
	}
	credentialsSecret := &corev1.Secret{}
	secretName := resources.GetOperatorCredentialsSecretName(c.Jenkins)
	logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins Client will be created using credentials in kuberentes Secret named : %+v", secretName))

	err = c.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: c.Jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while trying to get kuberentes Secret named : %+v: %s", secretName, err))
		return nil, stackerr.WithStack(err)
	}
	currentJenkinsMasterPod, err := c.GetJenkinsMasterPod()
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while trying to get GetJenkinsMasterPod : %s", err))
		return nil, err
	}
	var tokenCreationTime *time.Time
	tokenCreationTimeBytes := credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey]
	if tokenCreationTimeBytes != nil {
		tokenCreationTime = &time.Time{}
		err = tokenCreationTime.UnmarshalText(tokenCreationTimeBytes)
		if err != nil {
			tokenCreationTime = nil
		}
	}
	if credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] == nil ||
		tokenCreationTimeBytes == nil || tokenCreationTime == nil ||
		currentJenkinsMasterPod.ObjectMeta.CreationTimestamp.Time.UTC().After(tokenCreationTime.UTC()) {
		userName := string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
		jenkinsClient, err := jenkinsclient.NewUserAndPasswordAuthorization(
			jenkinsAPIUrl,
			userName,
			string(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey]))
		if err != nil {
			logger.V(log.VDebug).Info(fmt.Sprintf("Error while trying to create Jenkins Client with UserAndPasswordAuthorization: %s", err))
			return nil, err
		}

		token, err := jenkinsClient.GenerateToken(userName, "token")
		if err != nil {
			logger.V(log.VDebug).Info(fmt.Sprintf("Error while trying to generate token for Jenkins Client with: %s", err))
			return nil, err
		}

		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] = []byte(token.GetToken())
		now, _ := time.Now().UTC().MarshalText()
		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey] = now
		err = c.UpdateResource(credentialsSecret)
		if err != nil {
			logger.V(log.VDebug).Info(fmt.Sprintf("Error while updating credentialsSecret :%+v: %s", credentialsSecret.Name, err))
			return nil, stackerr.WithStack(err)
		}
	}
	return jenkinsclient.NewUserAndPasswordAuthorization(
		jenkinsAPIUrl,
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey]))
}

// GetJenkinsOpts gets JENKINS_OPTS env parameter, parses it's values and returns it as a map`
func GetJenkinsOpts(jenkins v1alpha2.Jenkins) map[string]string {
	envs := jenkins.Spec.Master.Containers[0].Env
	jenkinsOpts := make(map[string]string)

	for key, value := range envs {
		if value.Name == "JENKINS_OPTS" {
			jenkinsOptsEnv := envs[key]
			jenkinsOptsWithDashes := jenkinsOptsEnv.Value
			if len(jenkinsOptsWithDashes) == 0 {
				return nil
			}

			jenkinsOptsWithEqOperators := strings.Split(jenkinsOptsWithDashes, " ")

			for _, vx := range jenkinsOptsWithEqOperators {
				opt := strings.Split(vx, "=")
				jenkinsOpts[strings.ReplaceAll(opt[0], "--", "")] = opt[1]
			}

			return jenkinsOpts
		}
	}
	return nil
}
