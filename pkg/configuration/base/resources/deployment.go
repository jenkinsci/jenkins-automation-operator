package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// NewJenkinsMasterPod builds Jenkins Master Kubernetes Pod resource.
func NewJenkinsDeployment(objectMeta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins, jenkinsSpec *v1alpha2.JenkinsSpec) *appsv1.Deployment {
	serviceAccountName := objectMeta.Name
	objectMeta.Annotations = jenkinsSpec.Master.Annotations
	objectMeta.Name = GetJenkinsDeploymentName(jenkins)
	selector := &metav1.LabelSelector{MatchLabels: objectMeta.Labels}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectMeta.Name,
			Namespace: objectMeta.Namespace,
			Labels:    objectMeta.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objectMeta,
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					NodeSelector:       jenkinsSpec.Master.NodeSelector,
					InitContainers:     newInitContainers(jenkins, jenkinsSpec),
					Containers:         newContainers(jenkins, jenkinsSpec),
					Volumes:            append(GetJenkinsMasterPodBaseVolumes(jenkins), jenkinsSpec.Master.Volumes...),
					SecurityContext:    jenkinsSpec.Master.SecurityContext,
					ImagePullSecrets:   jenkinsSpec.Master.ImagePullSecrets,
					Tolerations:        jenkinsSpec.Master.Tolerations,
					PriorityClassName:  jenkinsSpec.Master.PriorityClassName,
				},
			},
			Selector: selector,
		},
	}
}

// GetJenkinsDeploymentName returns Jenkins deployment name for given CR
func GetJenkinsDeploymentName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("jenkins-%s", jenkins.Name)
}
