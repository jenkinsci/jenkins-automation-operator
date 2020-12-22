package base

import (
	"fmt"

	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	stackerr "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *JenkinsBaseConfigurationReconciler) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	r.logger.Info(fmt.Sprintf("Creating configMap: %s", configMap.Name))
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *JenkinsBaseConfigurationReconciler) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	r.logger.Info(fmt.Sprintf("Creating configMap: %s", configMap.Name))
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *JenkinsBaseConfigurationReconciler) createBasePluginsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewBasePluginConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	r.logger.Info(fmt.Sprintf("Creating configMap: %s", configMap.Name))
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}
