package exec

import (
	"os"
	"path/filepath"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubeExecClient struct {
	Client *rest.Config
}

var (
	logger = log.Log.WithName("exec")
)

func (e *KubeExecClient) InitKubeGoClient() error {
	var err error

	home := homedir.HomeDir()
	serviceHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	servicePort := os.Getenv("KUBERNETES_SERVICE_PORT")
	if serviceHost != "" && servicePort != "" {
		logger.Info("Using in-cluster configuration")
		e.Client, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			return err
		}
	} else if home != "" {
		logger.Info("Using local kubeconfig")
		e.Client, err = clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
		if err != nil {
			return err
		}
	}
	return nil
}
