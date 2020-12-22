package resources

import (
	"fmt"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
)

// ServiceKind the kind name for Service
const ServiceKind = "Service"

// UpdateService returns new service with override fields from config
func UpdateService(current corev1.Service, desired v1alpha2.Service) corev1.Service {
	current.ObjectMeta.Annotations = desired.Annotations
	for key, value := range desired.Labels {
		current.ObjectMeta.Labels[key] = value
	}
	current.Spec.Type = desired.Type
	current.Spec.LoadBalancerIP = desired.LoadBalancerIP
	current.Spec.LoadBalancerSourceRanges = desired.LoadBalancerSourceRanges
	if len(current.Spec.Ports) == 0 {
		current.Spec.Ports = []corev1.ServicePort{{}}
	}
	current.Spec.Ports[0].Port = desired.Port
	current.Spec.Ports[0].Name = desired.PortName
	if desired.NodePort != 0 {
		current.Spec.Ports[0].NodePort = desired.NodePort
	}

	return current
}

// GetJenkinsHTTPServiceName returns Kubernetes service name used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}

// GetJenkinsJNLPServiceName returns Kubernetes service name used for expose Jenkins JNLP endpoint
func GetJenkinsJNLPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s-jnlp", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}
