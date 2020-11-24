package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetBasePluginsVolumeNameConfigMapName returns name of Kubernetes config map used to init configuration
func GetBasePluginsVolumeNameConfigMapName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s-base-plugins", constants.LabelAppValue, jenkins.ObjectMeta.Name)
}

func getPluginsList(plugins []v1alpha2.Plugin) string {
	logger := log.WithName("jenkinsimage_getPluginsList")
	pluginsAsText := ""
	for _, v := range plugins {
		pluginsAsText += fmt.Sprintln(fmt.Sprintf(PluginDefinitionFormat, v.Name, v.Version))
		logger.Info(fmt.Sprintf("Adding plugin %s:%s ", v.Name, v.Version))
	}
	return pluginsAsText
}

// NewBasePluginConfigMap builds Kubernetes config map used for base plugins
func NewBasePluginConfigMap(meta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) (*corev1.ConfigMap, error) {
	meta.Name = GetBasePluginsVolumeNameConfigMapName(jenkins)
	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data: map[string]string{
			basePluginsFileName: getPluginsList(jenkins.Status.Spec.Master.BasePlugins),
		},
	}, nil
}
