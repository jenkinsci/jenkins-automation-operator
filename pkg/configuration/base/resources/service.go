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
func UpdateService(actual corev1.Service, config v1alpha2.Service) corev1.Service {
	actual.ObjectMeta.Annotations = config.Annotations
	for key, value := range config.Labels {
		actual.ObjectMeta.Labels[key] = value
	}
	actual.Spec.Type = config.Type
	actual.Spec.LoadBalancerIP = config.LoadBalancerIP
	actual.Spec.LoadBalancerSourceRanges = config.LoadBalancerSourceRanges
	if len(actual.Spec.Ports) == 0 {
		actual.Spec.Ports = []corev1.ServicePort{{}}
	}
	actual.Spec.Ports[0].Port = config.Port
	if config.NodePort != 0 {
		actual.Spec.Ports[0].NodePort = config.NodePort
	}

	return actual
}

// GetJenkinsHTTPServiceName returns Kubernetes service name used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}

// GetJenkinsJNLPServiceName returns Kubernetes service name used for expose Jenkins JNLP endpoint
func GetJenkinsJNLPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s-jnlp", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}
