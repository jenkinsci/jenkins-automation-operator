package base

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
					Container: v1alpha1.Container{
						Env: []v1.EnvVar{
							{
								Name:  "SOME_VALUE",
								Value: "",
							},
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
					Container: v1alpha1.Container{
						Env: []v1.EnvVar{
							{
								Name:  "JENKINS_HOME",
								Value: "",
							},
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

func TestValidateReservedVolumes(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "not-used-name",
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateReservedVolumes()
		assert.Equal(t, true, got)
	})
	t.Run("used reserved name", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: resources.JenkinsHomeVolumeName,
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateReservedVolumes()
		assert.Equal(t, false, got)
	})
}

func TestValidateContainerVolumeMounts(t *testing.T) {
	t.Run("default Jenkins master container", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Container)
		assert.Equal(t, true, got)
	})
	t.Run("one extra volume", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "example",
						},
					},
					Container: v1alpha1.Container{
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "example",
								MountPath: "/test",
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Container)
		assert.Equal(t, true, got)
	})
	t.Run("empty mountPath", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "example",
						},
					},
					Container: v1alpha1.Container{
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "example",
								MountPath: "", // empty
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Container)
		assert.Equal(t, false, got)
	})
	t.Run("missing volume", func(t *testing.T) {
		jenkins := v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
					Container: v1alpha1.Container{
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "missing-volume",
								MountPath: "/test",
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Container)
		assert.Equal(t, false, got)
	})
}

func TestValidateConfigMapVolume(t *testing.T) {
	namespace := "default"
	t.Run("optional", func(t *testing.T) {
		optional := true
		volume := corev1.Volume{
			Name: "name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			nil, false, false)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, required", func(t *testing.T) {
		optional := false
		configMap := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "configmap-name"}}
		jenkins := &v1alpha1.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMap.Name,
					},
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), &configMap)
		assert.NoError(t, err)
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("missing configmap", func(t *testing.T) {
		optional := false
		configMap := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "configmap-name"}}
		jenkins := &v1alpha1.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMap.Name,
					},
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.False(t, got)
	})
}

func TestValidateSecretVolume(t *testing.T) {
	namespace := "default"
	t.Run("optional", func(t *testing.T) {
		optional := true
		volume := corev1.Volume{
			Name: "name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional: &optional,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			nil, false, false)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, required", func(t *testing.T) {
		optional := false
		secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "secret-name"}}
		jenkins := &v1alpha1.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional:   &optional,
					SecretName: secret.Name,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), &secret)
		assert.NoError(t, err)
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("missing secret", func(t *testing.T) {
		optional := false
		secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "secret-name"}}
		jenkins := &v1alpha1.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional:   &optional,
					SecretName: secret.Name,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.False(t, got)
	})
}
