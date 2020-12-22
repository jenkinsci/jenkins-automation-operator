package resources

import (
	"context"
	"fmt"

	logx "github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger           = logx.Log
	EmptyListOptions = metav1.ListOptions{}
	EmptyGetOptions  = metav1.GetOptions{}
)

const (
	ServiceAccountNameAnnotation = "kubernetes.io/service-account.name"
	BuilderServiceAccountName    = "builder"
)

// GetDockerBuilderSecretName returns the *first* Docker secret used for pushing images into the openshift registry
// in the current namespace or empty string
func GetDockerBuilderSecretName(namespace string, clientSet client.Client) (string, error) {
	secrets := &corev1.SecretList{}
	err := clientSet.List(context.TODO(), secrets, client.InNamespace(namespace))
	if err != nil {
		logger.V(logx.VDebug).Info(fmt.Sprintf("Error while getting secret for JenkinsImage: %s ", namespace))
		return "", err
	}
	for _, secret := range secrets.Items {
		if secret.ObjectMeta.Annotations[ServiceAccountNameAnnotation] == BuilderServiceAccountName {
			return secret.Name, nil
		}
	}
	return "", nil
}
