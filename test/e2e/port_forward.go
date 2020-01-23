package e2e

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type portForwardToPodRequest struct {
	// config is the kubernetes config
	config *rest.Config
	// pod is the selected pod for this port forwarding
	pod v1.Pod
	// localPort is the local port that will be selected to expose the podPort
	localPort int
	// podPort is the target port for the pod
	podPort int
	// Steams configures where to write or read input from
	streams genericclioptions.IOStreams
	// stopCh is the channel used to manage the port forward lifecycle
	stopCh <-chan struct{}
	// readyCh communicates when the tunnel is ready to receive traffic
	readyCh chan struct{}
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	_ = l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func setupPortForwardToPod(t *testing.T, namespace, podName string, podPort int) (port int, cleanUpFunc func(), waitFunc func(), portForwardFunc func(), err error) {
	port, err = getFreePort()
	if err != nil {
		t.Fatal(err)
	}

	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// stopCh control the port forwarding lifecycle. When it gets closed the
	// port forward will terminate
	stopCh := make(chan struct{}, 1)
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	req := portForwardToPodRequest{
		config: framework.Global.KubeConfig,
		pod: v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		},
		localPort: port,
		podPort:   podPort,
		streams:   stream,
		stopCh:    stopCh,
		readyCh:   readyCh,
	}

	waitFunc = func() {
		t.Log("Waiting for the port-forward.")
		<-readyCh
		t.Log("The port-forward is established.")
	}

	portForwardFunc = func() {
		err := portForwardToPod(req)
		if err != nil {
			panic(err)
		}
	}

	cleanUpFunc = func() {
		t.Log("Closing port-forward")
		close(stopCh)
	}

	return
}

func portForwardToPod(req portForwardToPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.pod.Namespace, req.pod.Name)
	hostIP := strings.TrimLeft(req.config.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.config)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.localPort, req.podPort)}, req.stopCh, req.readyCh, req.streams.Out, req.streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}
