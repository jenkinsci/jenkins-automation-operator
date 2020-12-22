package resources

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO Remove this meta object stuff and use Jenkins instead
// NewResourceObjectMeta builds ObjectMeta for all Kubernetes resources created by operator
func NewResourceObjectMeta(jenkins *v1alpha2.Jenkins) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      GetResourceName(jenkins),
		Namespace: jenkins.ObjectMeta.Namespace,
		Labels:    BuildResourceLabels(jenkins),
	}
}

func GetExtraRoleBindingName(jenkins *v1alpha2.Jenkins, roleRef rbacv1.RoleRef) string {
	serviceAccountName := NewResourceObjectMeta(jenkins).Name
	var typeName string
	if roleRef.Kind == "ClusterRole" {
		typeName = "cr"
	} else {
		typeName = "r"
	}
	return fmt.Sprintf("%s-%s-%s", serviceAccountName, typeName, roleRef.Name)
}

// BuildResourceLabels returns labels for all Kubernetes resources created by operator
func BuildResourceLabels(jenkins *v1alpha2.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
	}
}

// GetResourceName returns name of Kubernetes resource base on Jenkins CR
func GetResourceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.Name)
}
