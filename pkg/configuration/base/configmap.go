package base

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	stackerr "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *JenkinsReconcilerBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	r.logger.Info("Creating NewScriptsConfigMap")
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *JenkinsReconcilerBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	r.logger.Info("Creating configMap")
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}
