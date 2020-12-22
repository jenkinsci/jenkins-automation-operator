package resources

import (
	"testing"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetJenkinsMasterPodBaseVolumes(t *testing.T) {
	t.Run("Jenkins master with base volumes", func(t *testing.T) {
		namespace := "default"
		jenkinsName := "example"
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}

		cmVolume, initVolume, secretVolume := checkSecretVolumesPresence(jenkins)

		assert.True(t, cmVolume)
		assert.True(t, initVolume)
		assert.True(t, secretVolume)
	})
}

func checkSecretVolumesPresence(jenkins *v1alpha2.Jenkins) (cmVolume, initVolume, secretVolume bool) {
	for _, volume := range GetJenkinsMasterPodBaseVolumes(jenkins) {
		switch volume.Name {
		case "scripts":
			cmVolume = true
		case "init-configuration":
			initVolume = true
		case "operator-credentials":
			secretVolume = true
		}
	}
	return cmVolume, initVolume, secretVolume
}
