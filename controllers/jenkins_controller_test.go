package controllers

import (
	"context"
	"fmt"
	"time"

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
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	JenkinsTestNamespace = "jenkins-operator-test"
	timeout              = time.Second * 30
	interval             = time.Millisecond * 250
)

var _ = Describe("Jenkins controller", func() {
	Logf("Starting test for Jenkins Controller")

	// Test creation of Jenkins with simple casc configuration
	Context("When Creating a Jenkins Instance with Default CASC and Backup Configured", func() {
		ctx := context.Background()
		jenkinsName := "jenkins-with-all"
		jenkins := GetJenkinsTestInstance(jenkinsName, JenkinsTestNamespace)

		It(fmt.Sprintf("Check Prerequisites (%s)", jenkinsName), func() {
			// Create Namespace for testing if not present
			CreateNamespaceIfNotPresent(ctx, JenkinsTestNamespace)
			// Create Edit Cluster Role if not in Openshift
			CreateEditClusterRoleIfNotPresent(ctx)
		})

		It(fmt.Sprintf("Create Test BackupVolume (%s)", jenkinsName), func() {
			// Create BackupVolume
			CreateBackupVolume(ctx, "test", JenkinsTestNamespace)
		})

		It(fmt.Sprintf("Create Test BackupStrategy (%s)", jenkinsName), func() {
			// Create BackupStrategy
			CreateBackupStrategy(ctx, "test", JenkinsTestNamespace)
		})

		It(fmt.Sprintf("Jenkins Should Be Created (%s)", jenkinsName), func() {
			// Create Jenkins instance
			ByCreatingJenkinsSuccesfully(ctx, jenkins)
			// Check if CR is created for Jenkins
			ByCheckingThatJenkinsExists(ctx, jenkins)
		})

		It(fmt.Sprintf("Default RoleBinding Should Be Created (%s)", jenkinsName), func() {
			// Check if default RoleBinding is created for Jenkins
			ByCheckingThatDefaultRoleBindingIsCreated(ctx, jenkins)
		})

		It(fmt.Sprintf("Deployment Should Be Created (%s)", jenkinsName), func() {
			// Check if Jenkins Deployment has been created
			ByCheckingThatTheDeploymentExists(ctx, jenkins)
		})

		It(fmt.Sprintf("Example Service Should Be Created (%s)", constants.DefaultService), func() {
			// Check if Jenkins Example Services is created
			ByCheckingThatServiceIsCreated(ctx, constants.DefaultService, JenkinsTestNamespace)
		})

		It(fmt.Sprintf("Example-Jnlp Service Should Be Created (%s)", constants.DefaultJnlpService), func() {
			// Check if Jenkins Jnlp Service is created
			ByCheckingThatServiceIsCreated(ctx, constants.DefaultJnlpService, JenkinsTestNamespace)
		})

		It(fmt.Sprintf(" Default CasC ConfigMap Should Be Created (%s)", jenkinsName), func() {
			// Check if Default CasC configuration is present for Jenkins
			ByCheckingThatConfigMapIsCreated(ctx, resources.JenkinsDefaultConfigMapName, JenkinsTestNamespace)
		})

		It(fmt.Sprintf("Jenkins PVC for persistence Should Be Created (%s)", jenkinsName), func() {
			// Check if PVC is present for Jenkins
			ByCheckingThatPVCIsCreated(ctx, jenkinsName, JenkinsTestNamespace)
		})

		It(fmt.Sprintf("BackupVolume PVC Should Be Created (%s)", jenkinsName), func() {
			// Check if PVC is present for Jenkins
			ByCheckingThatPVCIsCreated(ctx, "test-jenkins-backup", JenkinsTestNamespace)
		})

		// It(fmt.Sprintf("ServiceMonitor Should Be Created (%s)", jenkinsName), func() {
		//	// Check if ServiceMonitor is present for Jenkins
		//	ByCheckingThatServiceMonitorIsCreated(ctx, jenkinsName+"-monitored", JenkinsTestNamespace)
		// })

		It(fmt.Sprintf("Deployment Is Ready (%s)", jenkinsName), func() {
			// Check if Deployment is ready
			ByCheckingThatTheDeploymentIsReady(ctx, jenkins)
		})

		It(fmt.Sprintf("Namespace should be deleted (%s)", jenkinsName), func() {
			// Cleanup
			DeleteNamespaceIfPresent(ctx, JenkinsTestNamespace)
		})
	})
})

func CreateEditClusterRoleIfNotPresent(ctx context.Context) {
	editClusterRole := &rbacv1.ClusterRole{}
	nn := types.NamespacedName{Name: base.EditClusterRole, Namespace: JenkinsTestNamespace}
	err := k8sClient.Get(ctx, nn, editClusterRole)
	if err != nil {
		editClusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nn.Name,
				Namespace: nn.Namespace,
			},
			Rules: resources.NewDefaultPolicyRules(),
		}
		Expect(k8sClient.Create(ctx, editClusterRole)).Should(Succeed())
	}
}

func CreateNamespaceIfNotPresent(ctx context.Context, namespaceName string) {
	By("Creating Namespace if it does not exist")
	actual := &corev1.Namespace{}
	key := types.NamespacedName{Name: namespaceName}
	err := k8sClient.Get(ctx, key, actual)
	if err != nil {
		By("Create Namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	}
}

func CreateBackupVolume(ctx context.Context, name, namespace string) {
	By("Creating BackupVolume")
	actual := &v1alpha2.BackupVolume{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, key, actual)
	if err != nil {
		By("Create BackupVolume")
		bv := &v1alpha2.BackupVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha2.BackupVolumeSpec{
				Size: "1Gi",
			},
		}
		Expect(k8sClient.Create(ctx, bv)).Should(Succeed())
	}
}

func CreateBackupStrategy(ctx context.Context, name, namespace string) {
	By("Creating BackupStrategy")
	actual := &v1alpha2.BackupStrategy{}
	key := types.NamespacedName{Name: name}
	err := k8sClient.Get(ctx, key, actual)
	if err != nil {
		By("Create BackupStrategy")
		bs := &v1alpha2.BackupStrategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha2.BackupStrategySpec{
				Options: v1alpha2.BackupOptions{
					Jobs:    true,
					Plugins: true,
					Config:  true,
				},
				RestartAfterRestore: v1alpha2.RestartConfig{
					Enabled: true,
				},
				QuietDownDuringBackup: true,
			},
		}
		Expect(k8sClient.Create(ctx, bs)).Should(Succeed())
	}
}

func DeleteNamespaceIfPresent(ctx context.Context, namespaceName string) {
	By("Deleting Namespace if it is present")
	actual := &corev1.Namespace{}
	key := types.NamespacedName{Name: namespaceName}
	err := k8sClient.Get(ctx, key, actual)
	if err != nil {
		Fail(fmt.Sprintf("Error while deleting Namespace %s", namespaceName))
	}
	Expect(k8sClient.Delete(ctx, actual)).Should(Succeed())
}

func ByCheckingThatJenkinsExists(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("Checking that the Jenkins CR exists")
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

// func ByCheckingThatServiceMonitorIsCreated(ctx context.Context, name, namespace string) {
//	By("Checking that ServiceMonitor is created")
//	created := &monitoringv1.ServiceMonitor{}
//	key := types.NamespacedName{Namespace: namespace, Name: name}
//	actual := func() (*monitoringv1.ServiceMonitor, error) {
//		err := k8sClient.Get(ctx, key, created)
//		if err != nil {
//			return nil, err
//		}
//		return created, nil
//	}
//	Eventually(actual, timeout, interval).Should(Equal(created))
// }

func ByCheckingThatPVCIsCreated(ctx context.Context, name, namespace string) {
	By("Checking that PVC is created")
	created := &corev1.PersistentVolumeClaim{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	actual := func() (*corev1.PersistentVolumeClaim, error) {
		err := k8sClient.Get(ctx, key, created)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	Eventually(actual, timeout, interval).Should(Equal(created))
}

func ByCheckingThatServiceIsCreated(ctx context.Context, name, namespace string) {
	By("Checking that Service is created")
	created := &corev1.Service{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	actual := func() (*corev1.Service, error) {
		err := k8sClient.Get(ctx, key, created)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	Eventually(actual, timeout, interval).Should(Equal(created))
}

func ByCheckingThatConfigMapIsCreated(ctx context.Context, name, namespace string) {
	By("Checking that ConfigMap is created")
	created := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	actual := func() (*corev1.ConfigMap, error) {
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
	By("By checking that the Deployment exists")
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

func ByCheckingThatTheDeploymentIsReady(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By checking that the Deployment has enough replicas")
	expected := &appsv1.Deployment{}
	expectedName := resources.GetJenkinsDeploymentName(jenkins)
	key := types.NamespacedName{Namespace: jenkins.Namespace, Name: expectedName}
	actualReadyReplicas := func() (int32, error) {
		err := k8sClient.Get(ctx, key, expected)
		if err != nil {
			return 0, err
		}
		return expected.Status.ReadyReplicas, nil
	}
	Eventually(actualReadyReplicas, timeout, interval).Should(Equal(int32(1)))
}

func ByCreatingJenkinsSuccesfully(ctx context.Context, jenkins *v1alpha2.Jenkins) {
	By("By creating a new Jenkins")
	Expect(k8sClient.Create(ctx, jenkins)).Should(Succeed())
}

func GetJenkinsTestInstance(name string, namespace string) *v1alpha2.Jenkins {
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsSpec{
			ConfigurationAsCode: &v1alpha2.Configuration{
				Enabled:       true,
				DefaultConfig: true,
			},
			PersistentSpec: v1alpha2.JenkinsPersistentSpec{
				Enabled: true,
			},
			BackupVolumes: []string{"test"},
			// MetricsEnabled: true,
		},
	}
	return jenkins
}
