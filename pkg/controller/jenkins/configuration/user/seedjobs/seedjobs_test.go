package seedjobs

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestEnsureSeedJobs(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		// given
		logger := logf.ZapLogger(false)
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := jenkinsCustomResource()
		err = fakeClient.Create(ctx, jenkins)
		assert.NoError(t, err)

		agentName := "jnlp"
		secret := "test-secret"
		testNode := &gojenkins.Node{
			Raw: &gojenkins.NodeResponse{
				DisplayName: agentName,
			},
		}

		jobID := jenkins.Spec.SeedJobs[0].ID

		jenkinsClient.EXPECT().IsNotFoundError(nil).AnyTimes()
		jenkinsClient.EXPECT().GetJob(jobID).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(agentName).Return(secret, nil).AnyTimes()
		jenkinsClient.EXPECT().GetAllNodes().Return([]*gojenkins.Node{}, nil).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(agentName, 1, "The jenkins-operator generated agent", "/home/jenkins", agentName).Return(testNode, nil).AnyTimes()
		jenkinsClient.EXPECT().GetNode(agentName).Return(testNode, nil).AnyTimes()

		jenkinsClient.EXPECT().ExecuteScript(seedJobCreatingGroovyScript(jenkins.Spec.SeedJobs[0])).AnyTimes()

		seedJobClient := New(jenkinsClient, fakeClient, logger)

		_, err = seedJobClient.EnsureSeedJobs(jenkins)
		assert.NoError(t, err)

		_, err = jenkinsClient.GetJob(jobID)

		assert.False(t, jenkinsClient.IsNotFoundError(err))
	})

	t.Run("delete pod when no seed jobs", func(t *testing.T) {
		// given
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		namespace := "test-namespace"
		agentName := "test-agent"
		secret := "test-secret"
		jenkinsCustomRes := jenkinsCustomResource()
		jenkinsCustomRes.Spec.SeedJobs = []v1alpha2.SeedJob{}

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkinsClient.EXPECT().IsNotFoundError(nil).AnyTimes()
		jenkinsClient.EXPECT().GetNode(agentName).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(agentName).Return(secret, nil).AnyTimes()
		jenkinsClient.EXPECT().GetAllNodes().Return([]*gojenkins.Node{}, nil).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(agentName, 1, "The jenkins-operator generated agent", "/home/jenkins", agentName).AnyTimes()

		seedJobsClient := New(jenkinsClient, fakeClient, nil)

		err = fakeClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-deployment", agentName),
				Namespace: namespace,
			},
		})

		assert.NoError(t, err)

		// when
		_, err = seedJobsClient.EnsureSeedJobs(jenkinsCustomRes)
		assert.NoError(t, err)

		// then
		var deployment appsv1.Deployment
		err = fakeClient.Get(ctx, types.NamespacedName{Name: agentName, Namespace: namespace}, &deployment)

		if !errors.IsNotFound(err) {
			t.Fatal("deployment not removed")
		}
	})
}

func jenkinsCustomResource() *v1alpha2.Jenkins {
	return &v1alpha2.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "default",
		},
		Spec: v1alpha2.JenkinsSpec{
			Master: v1alpha2.JenkinsMaster{
				Annotations: map[string]string{"test": "label"},
				Containers: []v1alpha2.Container{
					{
						Name:  resources.JenkinsMasterContainerName,
						Image: "jenkins/jenkins",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("300m"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			SeedJobs: []v1alpha2.SeedJob{
				{
					ID:                    "jenkins-operator-e2e",
					JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
					Targets:               "cicd/jobs/*.jenkins",
					Description:           "Jenkins Operator e2e tests repository",
					RepositoryBranch:      "master",
					RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
				},
			},
		},
	}
}

func TestSeedJobs_isRecreatePodNeeded(t *testing.T) {
	seedJobsClient := New(nil, nil, nil)
	t.Run("empty", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.False(t, got)
	})
	t.Run("same", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.False(t, got)
	})
	t.Run("removed one", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name1",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name1", "name2"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.True(t, got)
	})
	t.Run("renamed one", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name1",
					},
					{
						ID: "name3",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name1", "name2"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.True(t, got)
	})
}

func TestCreateAgent(t *testing.T) {
	t.Run("not fail when deployment is available", func(t *testing.T) {
		// given
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		namespace := "test-namespace"
		agentName := "test-agent"
		secret := "test-secret"

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewFakeClient()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkinsClient.EXPECT().IsNotFoundError(nil).AnyTimes()
		jenkinsClient.EXPECT().GetNode(agentName).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(agentName).Return(secret, nil).AnyTimes()
		jenkinsClient.EXPECT().GetAllNodes().Return([]*gojenkins.Node{}, nil).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(agentName, 1, "The jenkins-operator generated agent", "/home/jenkins", agentName).AnyTimes()

		seedJobsClient := New(jenkinsClient, fakeClient, nil)

		// when
		err = fakeClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-deployment", agentName),
				Namespace: namespace,
			},
		})

		assert.NoError(t, err)

		// then
		err = seedJobsClient.createAgent(jenkinsClient, fakeClient, jenkinsCustomResource(), namespace, agentName)
		assert.NoError(t, err)
	})
}
