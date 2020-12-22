package configuration

import (
	"testing"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGetJenkinsOpts(t *testing.T) {
	t.Run("JENKINS_OPTS is uninitialized", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "", Value: ""},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)
		assert.Equal(t, 0, len(opts))
	})

	t.Run("JENKINS_OPTS is empty", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "JENKINS_OPTS", Value: ""},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)
		assert.Equal(t, 0, len(opts))
	})

	t.Run("JENKINS_OPTS have --prefix argument ", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins"},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "httpPort")
		assert.Contains(t, opts, "prefix")
		assert.Equal(t, opts["prefix"], "/jenkins")
	})

	t.Run("JENKINS_OPTS have --prefix and --httpPort argument", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins --httpPort=8080"},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 2, len(opts))

		assert.Contains(t, opts, "prefix")
		assert.Equal(t, opts["prefix"], "/jenkins")

		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "8080")
	})

	t.Run("JENKINS_OPTS have --httpPort argument", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "JENKINS_OPTS", Value: "--httpPort=8080"},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "prefix")
		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "8080")
	})

	t.Run("JENKINS_OPTS have --httpPort=--8080 argument", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					Master: &v1alpha2.JenkinsMaster{
						Containers: []v1alpha2.Container{
							{
								Env: []corev1.EnvVar{
									{Name: "JENKINS_OPTS", Value: "--httpPort=--8080"},
								},
							},
						},
					},
				},
			},
		}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "prefix")
		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "--8080")
	})
}
