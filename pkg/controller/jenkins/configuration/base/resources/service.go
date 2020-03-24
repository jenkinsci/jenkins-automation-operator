package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"

	"net"
	"strings"
)

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

// GetJenkinsServiceName returns Kubernetes service name used for expose Jenkins HTTP and Slaves endpoint
func GetJenkinsServiceName(jenkins *v1alpha2.Jenkins, serviceType string) string {
	return fmt.Sprintf("%s-%s-%s", constants.OperatorName, serviceType, jenkins.ObjectMeta.Name)
}

// GetJenkinsServiceFQDN returns Kubernetes service FQDN used for expose Jenkins HTTP and Slaves endpoint
func GetJenkinsServiceFQDN(jenkins *v1alpha2.Jenkins, serviceType string) string {
	clusterDomain := getClusterDomain()

	return fmt.Sprintf("%s-%s-%s.%s.svc.%s", constants.OperatorName, serviceType, jenkins.ObjectMeta.Name, jenkins.ObjectMeta.Namespace, clusterDomain)
}

// GetClusterDomain returns Kubernetes cluster domain, default to "cluster.local"
func getClusterDomain() string {
	apiSvc := "kubernetes.default.svc"

	clusterDomain := "cluster.local"

	cname, err := net.LookupCNAME(apiSvc)
	if err != nil {
		return clusterDomain
	}

	clusterDomain = strings.TrimPrefix(cname, "kubernetes.default.svc")
	clusterDomain = strings.TrimPrefix(clusterDomain, ".")
	clusterDomain = strings.TrimSuffix(clusterDomain, ".")

	return clusterDomain
}
