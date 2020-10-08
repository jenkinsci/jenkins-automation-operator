package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Define utility constants for object names and testing timeouts/durations and intervals.
const (
	// Name                  = "test-image"
	JenkinsImageName      = "test-jenkinsimage"
	JenkinsImageNamespace = "default"
	timeout               = time.Second * 30
	interval              = time.Millisecond * 1000
	// duration = time.Second * 10
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("JenkinsImage controller", func() {
	Context("When deleting Pod associated to JenkinsImage", func() {
		It("The Pod should be recreated", func() {
			Logf("Starting")
			ctx := context.Background()
			jenkinsImage := GetJenkinsImageTestInstance(JenkinsImageName, JenkinsImageNamespace)
			ByCreatingJenkinsImageSuccesfully(ctx, jenkinsImage)
			ByCheckingThatJenkinsImageExists(ctx, jenkinsImage)
			ByCheckingThatThePodExists(ctx, jenkinsImage)
			ByCheckingThatThePodIsRecreatedAfterDeletion(ctx, jenkinsImage)
		})
	})
})

func ByCheckingThatJenkinsImageExists(ctx context.Context, jenkinsImage *v1alpha2.JenkinsImage) {
	By("By checking that the JenkinsImage exists")
	created := &v1alpha2.JenkinsImage{}
	expectedName := jenkinsImage.Name
	key := types.NamespacedName{Namespace: jenkinsImage.Namespace, Name: expectedName}
	actual := func() (*v1alpha2.JenkinsImage, error) {
		err := k8sClient.Get(ctx, key, created)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	Eventually(actual, timeout, interval).Should(Equal(created))
}

func ByCheckingThatThePodExists(ctx context.Context, jenkinsImage *v1alpha2.JenkinsImage) {
	By("By checking that the Pod exists")
	expected := &corev1.Pod{}
	expectedName := fmt.Sprintf(resources.NameWithSuffixFormat, jenkinsImage.Name, resources.BuilderSuffix)
	key := types.NamespacedName{Namespace: jenkinsImage.Namespace, Name: expectedName}
	actual := func() (*corev1.Pod, error) {
		err := k8sClient.Get(ctx, key, expected)
		if err != nil {
			return nil, err
		}
		return expected, nil
	}
	Eventually(actual, timeout, interval).ShouldNot(BeNil())
	Eventually(actual, timeout, interval).Should(Equal(expected))
}

func ByCreatingJenkinsImageSuccesfully(ctx context.Context, jenkinsImage *v1alpha2.JenkinsImage) {
	By("By creating a new JenkinsImage")
	Expect(k8sClient.Create(ctx, jenkinsImage)).Should(Succeed())
}

func ByCheckingThatThePodIsRecreatedAfterDeletion(ctx context.Context, jenkinsImage *v1alpha2.JenkinsImage) {
	By("By checking that the Pod is recreated after deletion")
	before := &corev1.Pod{}
	expectedName := fmt.Sprintf(resources.NameWithSuffixFormat, jenkinsImage.Name, resources.BuilderSuffix)
	key := types.NamespacedName{Namespace: jenkinsImage.Namespace, Name: expectedName}
	err := k8sClient.Get(ctx, key, before)
	if err != nil {
		Fail(fmt.Sprintf("Error while trying to get Pod %s", key))
	}
	creationTimeStampBefore := before.CreationTimestamp

	Logf("Creation timestamp for Pod before deletion: %s", creationTimeStampBefore)
	err = k8sClient.Delete(ctx, before)
	if err != nil {
		Fail(fmt.Sprintf("Error while trying to delete Pod %s", key))
	}
	after := &corev1.Pod{}
	actual := func() (*corev1.Pod, error) {
		err := k8sClient.Get(ctx, key, after)
		if err != nil {
			return nil, err
		}
		return after, nil
	}
	Eventually(actual, timeout, interval).ShouldNot(BeNil())
	creationTimeStampAfter := after.CreationTimestamp
	if !creationTimeStampBefore.Before(&creationTimeStampAfter) {
		Fail(fmt.Sprintf("Creation timestamp after deletion is not after before: %s / %s", creationTimeStampBefore, creationTimeStampAfter))
	}
}

func GetJenkinsImageTestInstance(name string, namespace string) *v1alpha2.JenkinsImage {
	return &v1alpha2.JenkinsImage{
		TypeMeta: metav1.TypeMeta{
			Kind: "JenkinsImage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsImageSpec{
			Plugins: []v1alpha2.JenkinsPlugin{
				{
					Name:    "test",
					Version: "1.0.0",
				},
			},
		},
	}
}
