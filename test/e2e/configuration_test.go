package e2e

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	// base
	jenkins := createJenkinsCR(t, namespace)
	createDefaultLimitsForContainersInNamespace(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)

	verifyJenkinsMasterPodAttributes(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)
	verifyBasePlugins(t, client)

	// user
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	verifyJenkinsSeedJobs(t, client, jenkins)
}

func createDefaultLimitsForContainersInNamespace(t *testing.T, namespace string) {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					DefaultRequest: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Default: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}

	t.Logf("LimitRange %+v", *limitRange)
	if err := framework.Global.Client.Create(context.TODO(), limitRange, nil); err != nil {
		t.Fatal(err)
	}
}

func verifyJenkinsMasterPodAttributes(t *testing.T, jenkins *v1alpha1.Jenkins) {
	jenkinsPod := getJenkinsMasterPod(t, jenkins)
	jenkins = getJenkins(t, jenkins.Namespace, jenkins.Name)

	for key, value := range jenkins.Spec.Master.Annotations {
		if jenkinsPod.ObjectMeta.Annotations[key] != value {
			t.Fatalf("Invalid Jenkins pod annotation expected '%+v', actual '%+v'", jenkins.Spec.Master.Annotations, jenkinsPod.ObjectMeta.Annotations)
		}
	}

	if jenkinsPod.Spec.Containers[0].Image != jenkins.Spec.Master.Image {
		t.Fatalf("Invalid jenkins pod image expected '%s', actual '%s'", jenkins.Spec.Master.Image, jenkinsPod.Spec.Containers[0].Image)
	}

	if !reflect.DeepEqual(jenkinsPod.Spec.Containers[0].Resources, jenkins.Spec.Master.Resources) {
		t.Fatalf("Invalid jenkins pod continer resources expected '%+v', actual '%+v'", jenkins.Spec.Master.Resources, jenkinsPod.Spec.Containers[0].Resources)
	}

	t.Log("Jenkins pod attributes are valid")
}

func verifyBasePlugins(t *testing.T, jenkinsClient *gojenkins.Jenkins) {
	installedPlugins, err := jenkinsClient.GetPlugins(1)
	if err != nil {
		t.Fatal(err)
	}

	for rootPluginName, p := range plugins.BasePluginsMap {
		rootPlugin, err := plugins.New(rootPluginName)
		if err != nil {
			t.Fatal(err)
		}
		if found, ok := isPluginValid(installedPlugins, *rootPlugin); !ok {
			t.Fatalf("Invalid plugin '%s', actual '%+v'", rootPlugin, found)
		}
		for _, requiredPlugin := range p {
			if found, ok := isPluginValid(installedPlugins, requiredPlugin); !ok {
				t.Fatalf("Invalid plugin '%s', actual '%+v'", requiredPlugin, found)
			}
		}
	}

	t.Log("Base plugins have been installed")
}

func verifyJenkinsSeedJobs(t *testing.T, client *gojenkins.Jenkins, jenkins *v1alpha1.Jenkins) {
	t.Logf("Attempting to get configure seed job status '%v'", seedjobs.ConfigureSeedJobsName)

	configureSeedJobs, err := client.GetJob(seedjobs.ConfigureSeedJobsName)
	assert.NoError(t, err)
	assert.NotNil(t, configureSeedJobs)
	build, err := configureSeedJobs.GetLastSuccessfulBuild()
	assert.NoError(t, err)
	assert.NotNil(t, build)

	seedJobName := "jenkins-operator-configure-seed-job"
	t.Logf("Attempting to verify if seed job has been created '%v'", seedJobName)
	seedJob, err := client.GetJob(seedJobName)
	assert.NoError(t, err)
	assert.NotNil(t, seedJob)

	build, err = seedJob.GetLastSuccessfulBuild()
	assert.NoError(t, err)
	assert.NotNil(t, build)

	err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}, jenkins)
	assert.NoError(t, err, "couldn't get jenkins custom resource")
	assert.NotNil(t, jenkins.Status.Builds)
	assert.NotEmpty(t, jenkins.Status.Builds)

	jobCreatedByDSLPluginName := "build-jenkins-operator"
	err = wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to verify if job '%s' has been created ", jobCreatedByDSLPluginName)
		seedJob, err := client.GetJob(jobCreatedByDSLPluginName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(t, err)
}

func isPluginValid(plugins *gojenkins.Plugins, requiredPlugin plugins.Plugin) (*gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return p, false
	}

	if !p.Active || !p.Enabled || p.Deleted {
		return p, false
	}

	return p, requiredPlugin.Version == p.Version
}
