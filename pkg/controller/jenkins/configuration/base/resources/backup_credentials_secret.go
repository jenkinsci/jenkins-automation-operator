package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetBackupCredentialsSecretName returns name of Kubernetes secret used to store backup credentials
func GetBackupCredentialsSecretName(jenkins *v1alpha1.Jenkins) string {
	return fmt.Sprintf("%s-backup-credentials-%s", constants.OperatorName, jenkins.Name)
}

// NewBackupCredentialsSecret builds the Kubernetes secret used to store backup credentials
func NewBackupCredentialsSecret(jenkins *v1alpha1.Jenkins) *corev1.Secret {
	meta := metav1.ObjectMeta{
		Name:      GetBackupCredentialsSecretName(jenkins),
		Namespace: jenkins.ObjectMeta.Namespace,
		Labels:    BuildLabelsForWatchedResources(jenkins),
	}

	return &corev1.Secret{
		TypeMeta:   buildSecretTypeMeta(),
		ObjectMeta: meta,
	}
}
