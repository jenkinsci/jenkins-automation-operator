package controllers

import (
	"context"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	// Name                  = "test-image"
	JenkinsName        = "test-jenkins"
	MinimalJenkinsName = "minimal-jenkins"
	JenkinsNamespace   = "default"
)

var _ = Describe("Jenkins controller", func() {
	Logf("Starting")

	Context("When Creating a Minimal Jenkins Instance", func() {
		ctx := context.Background()
		jenkins := GetMinimalJenkinsTestInstance(MinimalJenkinsName, JenkinsNamespace)
		It("Deployment Should Be Created", func() {
			CreateEditClusterRole(ctx)
			ByCreatingJenkinsSuccesfully(ctx, jenkins)
			ByCheckingThatJenkinsExists(ctx, jenkins)
			ByCheckingThatDefaultRoleBindingIsCreated(ctx, jenkins)
			ByCheckingThatTheDeploymentExists(ctx, jenkins)
		})
	})

	Context("When Creating a Jenkins CR", func() {
		ctx := context.Background()
		jenkins := GetJenkinsTestInstance(JenkinsName, JenkinsNamespace)
		It("Deployment Should Be Created", func() {
			ByCreatingJenkinsSuccesfully(ctx, jenkins)
			ByCheckingThatJenkinsExists(ctx, jenkins)
			ByCheckingThatTheDeploymentExists(ctx, jenkins)
		})
	})

	Context("When Creating a Jenkins CR With specified Master.Image", func() {
		ctx := context.Background()
		image := "my-image"
		jenkinsWithImage := "jenkins-with-image"
		jenkins := GetJenkinsTestInstanceWithMasterImage(jenkinsWithImage, JenkinsNamespace, image)
		It("Deployment Should Be Created With That Image Name", func() {
			ByCreatingJenkinsSuccesfully(ctx, jenkins)
			ByCheckingThatJenkinsExists(ctx, jenkins)
			ByCheckingThatTheDeploymentExists(ctx, jenkins)
			ByCheckingThatDeploymentImageIs(ctx, jenkins)
		})
	})
})

func CreateEditClusterRole(ctx context.Context) {
	editClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      base.EditClusterRole,
			Namespace: JenkinsNamespace,
		},
		Rules: resources.NewDefaultPolicyRules(),
	}
	Expect(k8sClient.Create(ctx, editClusterRole)).Should(Succeed())
}

func ByCheckingThatDeploymentImageIs(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By checking that the Jenkins exists")
	created := &v1alpha2.Jenkins{}
	expectedName := jenkins.Name
	key := types.NamespacedName{Namespace: jenkins.Namespace, Name: expectedName}
	actual := func() (string, error) {
		err := k8sClient.Get(ctx, key, created)
		if err != nil {
			return "", err
		}
		return created.Spec.Master.Containers[0].Image, nil
	}
	Eventually(actual, timeout, interval).Should(Equal(jenkins.Spec.Master.Containers[0].Image))
}

func GetJenkinsTestInstanceWithMasterImage(jenkinsName string, namespaceName string, imageName string) *v1alpha2.Jenkins {
	jenkins := GetJenkinsTestInstance(jenkinsName, namespaceName)
	jenkins.Spec.Master.Containers[0].Image = imageName
	return jenkins
}

func ByCheckingThatJenkinsExists(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By checking that the Jenkins exists")
	created := &v1alpha2.Jenkins{}
	expectedName := jenkins.Name
	key := types.NamespacedName{Namespace: jenkins.Namespace, Name: expectedName}
	actual := func() (*v1alpha2.Jenkins, error) {
		err := k8sClient.Get(ctx, key, created)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	Eventually(actual, timeout, interval).Should(Equal(created))
}

func ByCheckingThatDefaultRoleBindingIsCreated(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By checking that default RoleBinding is created")
	expected := &rbacv1.RoleBinding{}
	roleRef := rbacv1.RoleRef{Name: "edit", Kind: "ClusterRole", APIGroup: base.AuthorizationAPIGroup}
	expectedName := resources.GetExtraRoleBindingName(jenkins, roleRef)
	key := types.NamespacedName{Namespace: jenkins.Namespace, Name: expectedName}
	actual := func() (*rbacv1.RoleBinding, error) {
		err := k8sClient.Get(ctx, key, expected)
		if err != nil {
			return nil, err
		}
		return expected, nil
	}
	Eventually(actual, timeout, interval).ShouldNot(BeNil())
	Eventually(actual, timeout, interval).Should(Equal(expected))
}

func ByCheckingThatTheDeploymentExists(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By checking that the Pod exists")
	expected := &appsv1.Deployment{}
	expectedName := resources.GetJenkinsDeploymentName(jenkins)
	key := types.NamespacedName{Namespace: jenkins.Namespace, Name: expectedName}
	actual := func() (*appsv1.Deployment, error) {
		err := k8sClient.Get(ctx, key, expected)
		if err != nil {
			return nil, err
		}
		return expected, nil
	}
	Eventually(actual, timeout, interval).ShouldNot(BeNil())
	Eventually(actual, timeout, interval).Should(Equal(expected))
}

func ByCreatingJenkinsSuccesfully(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By creating a new Jenkins")
	Expect(k8sClient.Create(ctx, jenkins)).Should(Succeed())
}

func GetJenkinsTestInstance(name string, namespace string) *v1alpha2.Jenkins {
	// TODO fix e2e to use deployment instead of pod
	annotations := map[string]string{"test": "label"}
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: v1alpha2.JenkinsSpec{
			//			JenkinsAPISettings: v1alpha2.JenkinsAPISettings{
			//				AuthorizationStrategy: v1alpha2.ServiceAccountAuthorizationStrategy,
			//			},
			Master: &v1alpha2.JenkinsMaster{
				Annotations: annotations,
				Containers:  getJenkinsContainers(),
				BasePlugins: getJenkinsPlugins(),
				Volumes:     getJenkinsVolumes(),
			},
			Service: getJenkinsServices(),
			Roles:   getJenkinsRoles(name),
		},
	}
	return jenkins
}

func GetMinimalJenkinsTestInstance(name string, namespace string) *v1alpha2.Jenkins {
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return jenkins
}

func getJenkinsRoles(resourceName string) []rbacv1.RoleRef {
	return []rbacv1.RoleRef{
		{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     resourceName,
		},
	}
}

func getJenkinsServices() v1alpha2.Service {
	return v1alpha2.Service{
		Type: corev1.ServiceTypeNodePort,
		Port: constants.DefaultHTTPPortInt32,
	}
}

func getJenkinsVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "plugins-cache",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func getJenkinsContainers() []v1alpha2.Container {
	return []v1alpha2.Container{
		{
			Name:           resources.JenkinsMasterContainerName,
			ReadinessProbe: getJenkinsProbe(),
			LivenessProbe:  getJenkinsProbe(),
			VolumeMounts:   getJenkinsVolumeMounts(),
		},
		{
			Name:  "envoyproxy",
			Image: "envoyproxy/envoy-alpine:v1.14.1",
		},
	}
}

func getJenkinsVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "plugins-cache",
			MountPath: "/usr/share/jenkins/ref/plugins",
		},
	}
}

func getJenkinsProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/login",
				Port:   intstr.FromString("http"),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: int32(80),
		TimeoutSeconds:      int32(4),
		FailureThreshold:    int32(10),
	}
}

func getJenkinsPlugins() []v1alpha2.Plugin {
	return []v1alpha2.Plugin{
		{Name: "configuration-as-code", Version: "1.42"},
		{Name: "configuration-as-code-groovy", Version: "1.1"},
		{Name: "git", Version: "4.2.2"},
		{Name: "job-dsl", Version: "1.77"},
		{Name: "kubernetes-credentials-provider", Version: "0.13"},
		{Name: "kubernetes", Version: "1.25.2"},
		{Name: "workflow-aggregator", Version: "2.6"},
		{Name: "workflow-job", Version: "2.40"},
	}
}
