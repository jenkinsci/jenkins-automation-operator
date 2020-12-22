package base

import (
	"context"
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// NewServiceMonitor returns a prometheus service monitor
func NewServiceMonitor(serviceMonitorName, namespace string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       monitoringv1.ServiceMonitorsKind,
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceMonitorName,
			Namespace: namespace,
		},
	}
}

// NewJenkinsServiceMonitor returns a prometheus service monitor for jenkins
func (r *JenkinsBaseConfigurationReconciler) NewJenkinsServiceMonitor(jenkins *v1alpha2.Jenkins) *monitoringv1.ServiceMonitor {
	meta := resources.NewResourceObjectMeta(jenkins)

	desired := NewServiceMonitor(fmt.Sprintf("%s-monitored", meta.Name), meta.Namespace)

	endpoint := monitoringv1.Endpoint{
		Port: "web",
		Path: "/prometheus/",
	}

	labelSelector := metav1.LabelSelector{MatchLabels: meta.Labels}

	desired.Spec = monitoringv1.ServiceMonitorSpec{
		JobLabel:  "monitor-jenkins",
		Endpoints: []monitoringv1.Endpoint{endpoint},
		Selector:  labelSelector,
		NamespaceSelector: monitoringv1.NamespaceSelector{
			MatchNames: []string{meta.Namespace},
		},
	}

	return desired
}

// createServiceMonitor creates the service monitor for jenkins
func (r *JenkinsBaseConfigurationReconciler) createServiceMonitor(jenkins *v1alpha2.Jenkins) error {
	desired := r.NewJenkinsServiceMonitor(jenkins)

	err := r.CreateResource(desired)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failure creating the jenkins ServiceMonitor: %v", err)
		}
		current := &monitoringv1.ServiceMonitor{}
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = r.Client.Get(context.TODO(), types.NamespacedName{Name: desired.Name, Namespace: jenkins.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("failed to get %q service monitor: %v", current.Name, err)
			}
			current.Labels = desired.Labels
			current.Spec = desired.Spec
			current.Annotations = desired.Annotations

			return r.UpdateResource(current)
		})
		r.logger.Info(fmt.Sprintf("service monitor %q updated successfully", desired.Name))
		return retryErr
	}
	return nil
}
