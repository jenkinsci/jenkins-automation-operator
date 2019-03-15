package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetUserConfigurationSecretNameFromJenkins returns name of Kubernetes secret used to store jenkins operator credentials
func GetUserConfigurationSecretNameFromJenkins(jenkins *v1alpha1.Jenkins) string {
	return fmt.Sprintf("%s-user-configuration-%s", constants.OperatorName, jenkins.Name)
}

// GetUserConfigurationSecretName returns name of Kubernetes secret used to store jenkins operator credentials
func GetUserConfigurationSecretName(jenkinsCRName string) string {
	return fmt.Sprintf("%s-user-configuration-%s", constants.OperatorName, jenkinsCRName)
}

// NewUserConfigurationSecret builds the Kubernetes secret resource which is used to store user sensitive data for Jenkins configuration
func NewUserConfigurationSecret(jenkins *v1alpha1.Jenkins) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: buildServiceTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetUserConfigurationSecretNameFromJenkins(jenkins),
			Namespace: jenkins.ObjectMeta.Namespace,
			Labels:    BuildLabelsForWatchedResources(*jenkins),
		},
	}
}
