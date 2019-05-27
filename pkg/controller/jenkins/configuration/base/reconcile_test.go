package base

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

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
		jenkins := &v1alpha1.Jenkins{}
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
		jenkins := &v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
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
		jenkins := &v1alpha1.Jenkins{
			Spec: v1alpha1.JenkinsSpec{
				Master: v1alpha1.JenkinsMaster{
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
