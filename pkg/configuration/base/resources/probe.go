package resources

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewSimpleProbe(uri string, port string, scheme corev1.URIScheme, initialDelaySeconds int32) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   uri,
				Port:   intstr.FromString(port),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: initialDelaySeconds,
	}
}

func NewProbe(uri string, port string, scheme corev1.URIScheme, initialDelaySeconds, timeoutSeconds, failureThreshold int32) *corev1.Probe {
	p := NewSimpleProbe(uri, port, scheme, initialDelaySeconds)
	p.TimeoutSeconds = timeoutSeconds
	p.FailureThreshold = failureThreshold
	return p
}
