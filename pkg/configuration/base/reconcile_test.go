package base

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/client"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

// compareVolumes returns true if Jenkins pod and Jenkins CR volumes are the same
func (r *JenkinsBaseConfigurationReconciler) compareVolumes(actualPod corev1.Pod) bool {
	var withoutServiceAccount []corev1.Volume
	for _, volume := range actualPod.Spec.Volumes {
		if !strings.HasPrefix(volume.Name, actualPod.Spec.ServiceAccountName) {
			withoutServiceAccount = append(withoutServiceAccount, volume)
		}
	}
	jenkins := r.Configuration.Jenkins
	return reflect.DeepEqual(
		append(resources.GetJenkinsMasterPodBaseVolumes(jenkins), jenkins.Status.Spec.Master.Volumes...),
		withoutServiceAccount,
	)
}

func TestCompareVolumes(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						Volumes: []corev1.Volume{},
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
		reconciler := New(configuration.Configuration{Jenkins: jenkins}, client.JenkinsAPIConnectionSettings{})

		got := reconciler.compareVolumes(pod)

		assert.True(t, got)
	})
	t.Run("different", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						Volumes: []corev1.Volume{{
							Name: "added",
						},
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
		reconciler := New(configuration.Configuration{Jenkins: jenkins}, client.JenkinsAPIConnectionSettings{})

		got := reconciler.compareVolumes(pod)

		assert.False(t, got)
	})
	t.Run("added one volume", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						Volumes: []corev1.Volume{{
							Name: "added",
						},
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
		reconciler := New(configuration.Configuration{Jenkins: jenkins}, client.JenkinsAPIConnectionSettings{})

		got := reconciler.compareVolumes(pod)

		assert.True(t, got)
	})
}

func TestJenkinsReconcilerBaseConfiguration_verifyPlugins(t *testing.T) {
	log.SetupLogger(true)

	t.Run("happy, empty base and user plugins", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						BasePlugins: []v1alpha2.Plugin{},
					},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name1", Version: "0.0.1"}},
					},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.1"}},
					},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master:              &v1alpha2.JenkinsMaster{},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name", Version: "0.0.1"}},
					},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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

	t.Run("missing base plugin", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: &v1alpha2.JenkinsStatus{
				Spec: &v1alpha2.JenkinsSpec{
					ConfigurationAsCode: &v1alpha2.Configuration{},
					Master: &v1alpha2.JenkinsMaster{
						BasePlugins: []v1alpha2.Plugin{{Name: "plugin-name1", Version: "0.0.2"}},
					},
				},
			},
		}
		r := JenkinsBaseConfigurationReconciler{
			logger: log.Log,
			Configuration: configuration.Configuration{
				Jenkins: jenkins,
			},
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

func Test_compareEnv(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var expected []corev1.EnvVar
		var actual []corev1.EnvVar

		got := compareEnv(expected, actual)

		assert.True(t, got)
	})
	t.Run("same", func(t *testing.T) {
		expected := []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		}
		actual := []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		}

		got := compareEnv(expected, actual)

		assert.True(t, got)
	})
	t.Run("with KUBERNETES envs", func(t *testing.T) {
		expected := []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		}
		actual := []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
			{
				Name:  "KUBERNETES_PORT_443_TCP_ADDR",
				Value: "KUBERNETES_PORT_443_TCP_ADDR",
			},
			{
				Name:  "KUBERNETES_PORT",
				Value: "KUBERNETES_PORT",
			},
			{
				Name:  "KUBERNETES_PORT_443_TCP",
				Value: "KUBERNETES_PORT_443_TCP",
			},
			{
				Name:  "KUBERNETES_SERVICE_HOST",
				Value: "KUBERNETES_SERVICE_HOST",
			},
		}

		got := compareEnv(expected, actual)

		assert.True(t, got)
	})
	t.Run("different", func(t *testing.T) {
		expected := []corev1.EnvVar{
			{
				Name:  "name",
				Value: "value",
			},
		}
		actual := []corev1.EnvVar{
			{
				Name:  "name2",
				Value: "value2",
			},
		}

		got := compareEnv(expected, actual)

		assert.False(t, got)
	})
}

func TestCompareMap(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		expectedAnnotations := map[string]string{}
		actualAnnotations := map[string]string{}

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.True(t, got)
	})
	t.Run("the same", func(t *testing.T) {
		expectedAnnotations := map[string]string{"one": "two"}
		actualAnnotations := expectedAnnotations

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.True(t, got)
	})
	t.Run("one extra annotation in pod", func(t *testing.T) {
		expectedAnnotations := map[string]string{"one": "two"}
		actualAnnotations := map[string]string{"one": "two", "three": "four"}

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.True(t, got)
	})
	t.Run("one missing annotation", func(t *testing.T) {
		expectedAnnotations := map[string]string{"one": "two"}
		actualAnnotations := map[string]string{"three": "four"}

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.False(t, got)
	})
	t.Run("one value is different", func(t *testing.T) {
		expectedAnnotations := map[string]string{"one": "two"}
		actualAnnotations := map[string]string{"one": "three"}

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.False(t, got)
	})
	t.Run("one missing annotation and one value is different", func(t *testing.T) {
		expectedAnnotations := map[string]string{"one": "two", "missing": "something"}
		actualAnnotations := map[string]string{"one": "three"}

		got := compareMap(expectedAnnotations, actualAnnotations)

		assert.False(t, got)
	})
}

func TestCompareImagePullSecrets(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var expected []corev1.LocalObjectReference
		var actual []corev1.LocalObjectReference

		got := compareImagePullSecrets(expected, actual)

		assert.True(t, got)
	})
	t.Run("the same", func(t *testing.T) {
		expected := []corev1.LocalObjectReference{{Name: "test"}}
		actual := []corev1.LocalObjectReference{{Name: "test"}}

		got := compareImagePullSecrets(expected, actual)

		assert.True(t, got)
	})
	t.Run("one extra pull secret in pod", func(t *testing.T) {
		var expected []corev1.LocalObjectReference
		actual := []corev1.LocalObjectReference{{Name: "test"}}

		got := compareImagePullSecrets(expected, actual)

		assert.True(t, got)
	})
	t.Run("one missing image pull secret", func(t *testing.T) {
		expected := []corev1.LocalObjectReference{{Name: "test"}}
		var actual []corev1.LocalObjectReference

		got := compareImagePullSecrets(expected, actual)

		assert.False(t, got)
	})
}

func TestEnsureExtraRBAC(t *testing.T) {
	namespace := "default"
	jenkinsName := "example"
	log.SetupLogger(true)

	fetchAllRoleBindings := func(client k8sclient.Client) (roleBindings *rbacv1.RoleBindingList, err error) {
		roleBindings = &rbacv1.RoleBindingList{}
		err = client.List(context.TODO(), roleBindings, k8sclient.InNamespace(namespace))
		return
	}

	t.Run("empty", func(t *testing.T) {
		// given
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
			Spec: v1alpha2.JenkinsSpec{
				Roles: []rbacv1.RoleRef{},
			},
		}
		jenkins.Status = &v1alpha2.JenkinsStatus{
			Spec: jenkins.Spec.DeepCopy(),
		}
		config := configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
			Scheme:  scheme.Scheme,
		}
		reconcilier := New(config, client.JenkinsAPIConnectionSettings{})
		metaObject := resources.NewResourceObjectMeta(jenkins)

		// when
		err = reconcilier.createRBAC(jenkins)
		assert.NoError(t, err)
		err = reconcilier.ensureExtraRBACArePresent()
		assert.NoError(t, err)

		// then
		roleBindings, err := fetchAllRoleBindings(fakeClient)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(roleBindings.Items))
		assert.Equal(t, metaObject.Name, roleBindings.Items[0].Name)
	})
	clusterRoleKind := "ClusterRole"
	t.Run("one extra", func(t *testing.T) {
		// given
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
			Spec: v1alpha2.JenkinsSpec{
				Roles: []rbacv1.RoleRef{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     clusterRoleKind,
						Name:     "edit",
					},
				},
			},
		}
		jenkins.Status = &v1alpha2.JenkinsStatus{
			Spec: jenkins.Spec.DeepCopy(),
		}
		config := configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
			Scheme:  scheme.Scheme,
		}
		reconciler := New(config, client.JenkinsAPIConnectionSettings{})
		metaObject := resources.NewResourceObjectMeta(jenkins)

		// when
		err = reconciler.createRBAC(jenkins)
		assert.NoError(t, err)
		err = reconciler.ensureExtraRBACArePresent()
		assert.NoError(t, err)

		// then
		roleBindings, err := fetchAllRoleBindings(fakeClient)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(roleBindings.Items))
		assert.Equal(t, metaObject.Name, roleBindings.Items[0].Name)
		assert.Equal(t, jenkins.Spec.Roles[0], roleBindings.Items[1].RoleRef)
	})
	t.Run("two extra", func(t *testing.T) {
		// given
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
			Spec: v1alpha2.JenkinsSpec{
				Roles: []rbacv1.RoleRef{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     clusterRoleKind,
						Name:     "admin",
					},
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     clusterRoleKind,
						Name:     "edit",
					},
				},
			},
		}
		jenkins.Status = &v1alpha2.JenkinsStatus{
			Spec: jenkins.Spec.DeepCopy(),
		}
		config := configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
			Scheme:  scheme.Scheme,
		}
		reconciler := New(config, client.JenkinsAPIConnectionSettings{})
		metaObject := resources.NewResourceObjectMeta(jenkins)

		// when
		err = reconciler.createRBAC(jenkins)
		assert.NoError(t, err)
		err = reconciler.ensureExtraRBACArePresent()
		assert.NoError(t, err)

		// then
		roleBindings, err := fetchAllRoleBindings(fakeClient)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(roleBindings.Items))
		assert.Equal(t, metaObject.Name, roleBindings.Items[0].Name)
		assert.Equal(t, jenkins.Spec.Roles[0], roleBindings.Items[1].RoleRef)
		assert.Equal(t, jenkins.Spec.Roles[1], roleBindings.Items[2].RoleRef)
	})
	t.Run("delete one extra", func(t *testing.T) {
		// given
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
			Spec: v1alpha2.JenkinsSpec{
				Roles: []rbacv1.RoleRef{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     clusterRoleKind,
						Name:     "admin",
					},
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     clusterRoleKind,
						Name:     "edit",
					},
				},
			},
		}
		jenkins.Status = &v1alpha2.JenkinsStatus{
			Spec: jenkins.Spec.DeepCopy(),
		}
		config := configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
			Scheme:  scheme.Scheme,
		}
		reconciler := New(config, client.JenkinsAPIConnectionSettings{})
		metaObject := resources.NewResourceObjectMeta(jenkins)

		// when
		roleBindingSkipMe := resources.NewRoleBinding(jenkins, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     clusterRoleKind,
			Name:     "edit",
		})
		err = reconciler.CreateOrUpdateResource(roleBindingSkipMe)
		assert.NoError(t, err)
		err = reconciler.createRBAC(jenkins)
		assert.NoError(t, err)
		err = reconciler.ensureExtraRBACArePresent()
		assert.NoError(t, err)
		jenkins.Spec.Roles = []rbacv1.RoleRef{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     clusterRoleKind,
				Name:     "admin",
			},
		}
		err = reconciler.ensureExtraRBACArePresent()
		assert.NoError(t, err)

		// then
		roleBindings, err := fetchAllRoleBindings(fakeClient)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(roleBindings.Items))
		assert.Equal(t, metaObject.Name, roleBindings.Items[1].Name)
		assert.Equal(t, jenkins.Spec.Roles[0], roleBindings.Items[2].RoleRef)
	})
}

func TestCompareContainerResources(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var expected corev1.ResourceRequirements
		var actual corev1.ResourceRequirements

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("expected resources empty, actual resources set", func(t *testing.T) {
		var expected corev1.ResourceRequirements
		actual := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("request CPU the same values", func(t *testing.T) {
		actual := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("50m"),
			},
		}
		expected := actual

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("request memory the same values", func(t *testing.T) {
		actual := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		}
		expected := actual

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("limit CPU the same values", func(t *testing.T) {
		actual := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("50m"),
			},
		}
		expected := actual

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("limit memory the same values", func(t *testing.T) {
		actual := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		}
		expected := actual

		got := compareContainerResources(expected, actual)

		assert.True(t, got)
	})
	t.Run("request CPU different values", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("5m"),
			},
		}
		actual := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("50m"),
			},
		}

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("request memory different values", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5Mi"),
			},
		}
		actual := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		}

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("limit CPU different values", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("5m"),
			},
		}
		actual := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("50m"),
			},
		}

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("limit memory different values", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5Mi"),
			},
		}
		actual := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		}

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("request CPU not set in actual", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("5m"),
			},
		}
		var actual corev1.ResourceRequirements

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("request memory not set in actual", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5Mi"),
			},
		}
		var actual corev1.ResourceRequirements

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("limit CPU not set in actual", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("5m"),
			},
		}
		var actual corev1.ResourceRequirements

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
	t.Run("limit memory not set in actual", func(t *testing.T) {
		expected := corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5Mi"),
			},
		}
		var actual corev1.ResourceRequirements

		got := compareContainerResources(expected, actual)

		assert.False(t, got)
	})
}
