package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const configureTheme = `
import jenkins.*
import jenkins.model.*
import hudson.*
import hudson.model.*
import org.jenkinsci.plugins.simpletheme.ThemeElement
import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

Jenkins jenkins = Jenkins.getInstance()

def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

List<ThemeElement> configElements = new ArrayList<>();
configElements.add(new CssTextThemeElement("DEFAULT"));
configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
decorator.setElements(configElements);
decorator.save();

jenkins.save()
`

// GetUserConfigurationConfigMapNameFromJenkins returns name of Kubernetes config map used to user configuration
func GetUserConfigurationConfigMapNameFromJenkins(jenkins *v1alpha1.Jenkins) string {
	return fmt.Sprintf("%s-user-configuration-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// GetUserConfigurationConfigMapName returns name of Kubernetes config map used to user configuration
func GetUserConfigurationConfigMapName(jenkinsCRName string) string {
	return fmt.Sprintf("%s-user-configuration-%s", constants.OperatorName, jenkinsCRName)
}

// NewUserConfigurationConfigMap builds Kubernetes config map used to user configuration
func NewUserConfigurationConfigMap(jenkins *v1alpha1.Jenkins) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: buildConfigMapTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetUserConfigurationConfigMapNameFromJenkins(jenkins),
			Namespace: jenkins.ObjectMeta.Namespace,
			Labels:    BuildLabelsForWatchedResources(*jenkins),
		},
		Data: map[string]string{
			"1-configure-theme.groovy": configureTheme,
		},
	}
}
