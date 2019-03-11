package base

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
		nil, false, false)
	t.Run("happy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {
				"valid-plugin-name:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})
	t.Run("fail, no version in plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name": {
				"invalid-plugin-name",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("fail, no version in root plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name": {
				"invalid-plugin-name:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("fail, no version in plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name:1.0": {
				"invalid-plugin-name",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("happy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {
				"valid-plugin-name:1.0",
				"valid-plugin-name2:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})
	t.Run("hapy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})

}

func TestValidateJenkinsMasterPodEnvs(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Env: []v1.EnvVar{
						{
							Name:  "SOME_VALUE",
							Value: "",
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()
		assert.Equal(t, true, got)
	})
	t.Run("override JENKINS_HOME env", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Env: []v1.EnvVar{
						{
							Name:  "JENKINS_HOME",
							Value: "",
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()
		assert.Equal(t, false, got)
	})
}
