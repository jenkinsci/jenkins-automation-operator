package resources

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetJenkinsMasterPodBaseVolumes(t *testing.T) {
	t.Run("casc and groovy script with different configMap names", func(t *testing.T) {
		configMapName := "config-map"
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				ConfigurationAsCode:v1alpha2.ConfigurationAsCode{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: configMapName,
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "casc-script",
						},
					},
				},
				GroovyScripts:v1alpha2.GroovyScripts{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: configMapName,
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "groovy-script",
						},
					},
				},
			},
		}

		groovyExists := false
		cascExists := false

		for _, volume := range GetJenkinsMasterPodBaseVolumes(jenkins) {
			if volume.Name == ("gs-" + jenkins.Spec.GroovyScripts.Secret.Name) {
				groovyExists = true
			} else if volume.Name == ("casc-" + jenkins.Spec.ConfigurationAsCode.Secret.Name) {
				cascExists = true
			}
		}

		assert.True(t, groovyExists && cascExists)
	})
}
