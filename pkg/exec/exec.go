package exec

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	corev1 "k8s.io/api/core/v1"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

type KubeExecClient interface {
	InitKubeGoClient() error
	MakeRequest(*corev1.Pod, string, string) error
}

var (
	logger = log.Log.WithName("exec")
)

var _ KubeExecClient = (*kubeExecClient)(nil)

type kubeExecClient struct {
	client       *rest.Config
	execComplete chan bool
	execErr      chan error
}

func NewKubeExecClient() KubeExecClient {
	return &kubeExecClient{}
}

func (e *kubeExecClient) InitKubeGoClient() error {
	var err error
	// Initialize channels for managing exec goroutines
	e.execComplete = make(chan bool, 1)
	e.execErr = make(chan error, 1)

	// Initialize go-client client
	home := homedir.HomeDir()
	serviceHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	servicePort := os.Getenv("KUBERNETES_SERVICE_PORT")
	if serviceHost != "" && servicePort != "" {
		logger.Info("Using in-cluster configuration")
		e.client, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			return err
		}
	} else if home != "" {
		logger.Info("Using local kubeconfig")
		e.client, err = clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *kubeExecClient) MakeRequest(jenkinsPod *corev1.Pod, resourceName, script string) error {
	request, err := e.newScriptRequest(jenkinsPod, script)
	if err != nil {
		return err
	}
	err = e.runPodExec(request, resourceName)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Script %s for instance %s has been successful", script, resourceName))
	return nil
}

func (e *kubeExecClient) newScriptRequest(jenkinsPod *corev1.Pod, script string) (*rest.Request, error) {
	client, err := clientgocorev1.NewForConfig(e.client)
	if err != nil {
		return nil, err
	}

	podExecRequest := client.RESTClient().Post().Resource("pods").
		Name(jenkinsPod.Name).
		Namespace(jenkinsPod.Namespace).
		SubResource("exec")
	podExecOptions := &corev1.PodExecOptions{
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
		Container: "backup",
		Command: []string{
			"sh", "-c", script,
		},
	}
	logger.Info(strings.Join([]string{
		"sh", "-c", script,
	}, " "))

	podExecRequest.VersionedParams(podExecOptions, scheme.ParameterCodec)
	return podExecRequest, err
}

func (e *kubeExecClient) runPodExec(podExecRequest *rest.Request, resourceName string) error {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	remoteCommand, err := remotecommand.NewSPDYExecutor(e.client, "POST", podExecRequest.URL())
	if err != nil {
		logger.Info(fmt.Sprintf("Error while creating remote executor for backup %s", err.Error()))
		return err
	}

	streamOptions := remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	}
	go func() {
		err = remoteCommand.Stream(streamOptions)
		e.execErr <- err
		e.execComplete <- true
	}()
	// Running each exec in a goroutine as goclient REST SPDYRExecutor stream does not cancel
	// https://github.com/kubernetes/client-go/issues/554#issuecomment-578886198
	<-e.execComplete
	err = <-e.execErr
	if err != nil {
		logger.Info(fmt.Sprintf("Error while executing script %s", err.Error()))
		logger.Info(fmt.Sprintf("'%s' Execution STDERR\n\t%s", resourceName, stderr.String()))
		return err
	}
	logger.Info(fmt.Sprintf("'%s' Execution STDOUT\n\t%s", resourceName, stdout.String()))

	return err
}
