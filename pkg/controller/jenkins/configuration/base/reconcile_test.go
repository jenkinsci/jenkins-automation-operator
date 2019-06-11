package base

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCompareContainerVolumeMounts(t *testing.T) {
	t.Run("happy with service account", func(t *testing.T) {
		expectedContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-name",
					MountPath: "/mount/path",
				},
			},
		}
		actualContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-name",
					MountPath: "/mount/path",
				},
				{
					Name:      "jenkins-operator-example-token-dh4r9",
					MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					ReadOnly:  true,
				},
			},
		}

		got := CompareContainerVolumeMounts(expectedContainer, actualContainer)

		assert.True(t, got)
	})
	t.Run("happy without service account", func(t *testing.T) {
		expectedContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-name",
					MountPath: "/mount/path",
				},
			},
		}
		actualContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-name",
					MountPath: "/mount/path",
				},
			},
		}

		got := CompareContainerVolumeMounts(expectedContainer, actualContainer)

		assert.True(t, got)
	})
	t.Run("different volume mounts", func(t *testing.T) {
		expectedContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-name",
					MountPath: "/mount/path",
				},
			},
		}
		actualContainer := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "jenkins-operator-example-token-dh4r9",
					MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					ReadOnly:  true,
				},
			},
		}

		got := CompareContainerVolumeMounts(expectedContainer, actualContainer)

		assert.False(t, got)
	})
}

func TestCompareVolumes(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				ServiceAccountName: "service-account-name",
				Volumes:            resources.GetJenkinsMasterPodBaseVolumes(jenkins),
			},
		}
		reconciler := New(nil, nil, nil, jenkins, false, false)

		got := reconciler.compareVolumes(pod)

		assert.True(t, got)
	})
	t.Run("different", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []corev1.Volume{
						{
							Name: "added",
						},
					},
				},
			},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				ServiceAccountName: "service-account-name",
				Volumes:            resources.GetJenkinsMasterPodBaseVolumes(jenkins),
			},
		}
		reconciler := New(nil, nil, nil, jenkins, false, false)

		got := reconciler.compareVolumes(pod)

		assert.False(t, got)
	})
	t.Run("added one volume", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []corev1.Volume{
						{
							Name: "added",
						},
					},
				},
			},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				ServiceAccountName: "service-account-name",
				Volumes:            append(resources.GetJenkinsMasterPodBaseVolumes(jenkins), corev1.Volume{Name: "added"}),
			},
		}
		reconciler := New(nil, nil, nil, jenkins, false, false)

		got := reconciler.compareVolumes(pod)

		assert.True(t, got)
	})
}

func TestReconcileJenkinsBaseConfiguration_verifyPlugins(t *testing.T) {
	log.SetupLogger(true)

	t.Run("happy, empty base and user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, not empty base and user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name1", Version: "0.0.1"}},
					Plugins:     []v1alpha2.Plugin{{Name: "plugin-name2", Version: "0.0.1"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{
					{
						ShortName: "plugin-name1",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.1",
					},
					{
						ShortName: "plugin-name2",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.1",
					},
				},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, not empty base and empty user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.1"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{
					{
						ShortName: "plugin-name",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.1",
					},
				},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, empty base and not empty user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Plugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.1"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{
					{
						ShortName: "plugin-name",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.1",
					},
				},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("happy, plugin version matter for base plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.1"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{
					{
						ShortName: "plugin-name",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.2",
					},
				},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("plugin version matter for user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Plugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.2"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{
					{
						ShortName: "plugin-name",
						Active:    true,
						Deleted:   false,
						Enabled:   true,
						Version:   "0.0.1",
					},
				},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("missing base plugin", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.2"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("missing user plugin", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Plugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.2"}},
				},
			},
		}
		r := ReconcileJenkinsBaseConfiguration{
			logger:  log.Log,
			jenkins: jenkins,
		}
		pluginsInJenkins := &gojenkins.Plugins{
			Raw: &gojenkins.PluginResponse{
				Plugins: []gojenkins.Plugin{},
			},
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := client.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().GetPlugins(fetchAllPlugins).Return(pluginsInJenkins, nil)

		got, err := r.verifyPlugins(jenkinsClient)

		assert.NoError(t, err)
		assert.False(t, got)
	})
}
