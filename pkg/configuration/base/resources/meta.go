package resources

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewResourceObjectMeta builds ObjectMeta for all Kubernetes resources created by operator
func NewResourceObjectMeta(jenkins *v1alpha2.Jenkins) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      GetResourceName(jenkins),
		Namespace: jenkins.ObjectMeta.Namespace,
		Labels:    BuildResourceLabels(jenkins),
	}
}

// BuildResourceLabels returns labels for all Kubernetes resources created by operator
func BuildResourceLabels(jenkins *v1alpha2.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
	}
}

// BuildLabelsForWatchedResources returns labels for Kubernetes resources which operator want to watch
// resources with that labels should not be deleted after Jenkins CR deletion, to prevent this situation don't set
// any owner
func BuildLabelsForWatchedResources(crName string) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: crName,
		constants.LabelWatchKey:     constants.LabelWatchValue,
	}
}

// GetResourceName returns name of Kubernetes resource base on Jenkins CR
func GetResourceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}

// VerifyIfLabelsAreSet check is selected labels are set for specific resource
func VerifyIfLabelsAreSet(object metav1.Object, requiredLabels map[string]string) bool {
	for key, value := range requiredLabels {
		if object.GetLabels()[key] != value {
			return false
		}
	}

	return true
}

// Add labels to watched secrets
func AddLabelToWatchedSecrets(crName, secretName, namespace string, k8sClient k8s.Client) error {
	labelsForWatchedResources := BuildLabelsForWatchedResources(crName)

	if len(secretName) > 0 {
		secret := &corev1.Secret{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
		if err != nil {
			return err
		}

		if !VerifyIfLabelsAreSet(secret, labelsForWatchedResources) {
			if len(secret.ObjectMeta.Labels) == 0 {
				secret.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				secret.ObjectMeta.Labels[key] = value
			}

			if err = k8sClient.Update(context.TODO(), secret); err != nil {
				return err
			}
		}
	}
	return nil
}

// Adds label to watched configmaps
func AddLabelToWatchedCMs(crName, namespace string, k8sClient k8s.Client, configmaps []v1alpha2.ConfigMapRef) error {
	labelsForWatchedResources := BuildLabelsForWatchedResources(crName)

	for _, configMapRef := range configmaps {
		configMap := &corev1.ConfigMap{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: namespace}, configMap)
		if err != nil {
			return err
		}

		if !VerifyIfLabelsAreSet(configMap, labelsForWatchedResources) {
			if len(configMap.ObjectMeta.Labels) == 0 {
				configMap.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				configMap.ObjectMeta.Labels[key] = value
			}

			if err = k8sClient.Update(context.TODO(), configMap); err != nil {
				return err
			}
		}
	}
	return nil
}
