package base

import (
	"context"
	"fmt"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"

	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	oauthAnnotationKey     = "serviceaccounts.openshift.io/oauth-redirectreference.jenkins"
	oauthAnnotationPattern = "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}"
)

func (r *JenkinsBaseConfigurationReconciler) createServiceAccount(jenkins *v1alpha2.Jenkins) error {
	meta := resources.NewResourceObjectMeta(jenkins)
	serviceAccount := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: meta.Name, Namespace: meta.Namespace}, serviceAccount)
	annotations := r.Configuration.Jenkins.Spec.ServiceAccount.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	oauthAnnotationValue := fmt.Sprintf(oauthAnnotationPattern, meta.Name)
	annotations[oauthAnnotationKey] = oauthAnnotationValue

	msg := fmt.Sprintf("createServiceAccount with annotations %v", annotations)
	r.logger.V(log.VDebug).Info(msg)
	if err != nil && apierrors.IsNotFound(err) {
		serviceAccount = resources.NewServiceAccount(meta, annotations)
		if err = r.CreateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	if !compareMap(annotations, serviceAccount.Annotations) {
		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = map[string]string{}
		}
		for key, value := range annotations {
			serviceAccount.Annotations[key] = value
		}
		if err = r.UpdateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	}

	return nil
}
