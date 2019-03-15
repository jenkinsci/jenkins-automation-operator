package e2e

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkinsCRName := "e2e"
	numberOfExecutors := 6
	systemMessage := "Configuration as Code integration works!!!"
	systemMessageEnvName := "SYSTEM_MESSAGE"
	mySeedJob := seedJobConfig{
		SeedJob: v1alpha1.SeedJob{
			ID:                    "jenkins-operator",
			CredentialID:          "jenkins-operator",
			JenkinsCredentialType: v1alpha1.NoJenkinsCredentialCredentialType,
			Targets:               "cicd/jobs/*.jenkins",
			Description:           "Jenkins Operator repository",
			RepositoryBranch:      "master",
			RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
		},
	}

	// base
	createUserConfigurationSecret(t, jenkinsCRName, namespace, systemMessageEnvName, systemMessage)
	createUserConfigurationConfigMap(t, jenkinsCRName, namespace, numberOfExecutors, fmt.Sprintf("${%s}", systemMessageEnvName))
	jenkins := createJenkinsCR(t, jenkinsCRName, namespace, &[]v1alpha1.SeedJob{mySeedJob.SeedJob})
	createDefaultLimitsForContainersInNamespace(t, namespace)
	createKubernetesCredentialsProviderSecret(t, namespace, mySeedJob)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)

	verifyJenkinsMasterPodAttributes(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)
	verifyPlugins(t, client, jenkins)

	// user
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	verifyUserConfiguration(t, client, numberOfExecutors, systemMessage)
	verifyJenkinsSeedJobs(t, client, []seedJobConfig{mySeedJob})
}

func createUserConfigurationSecret(t *testing.T, jenkinsCRName string, namespace string, systemMessageEnvName, systemMessage string) {
	userConfiguration := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GetUserConfigurationSecretName(jenkinsCRName),
			Namespace: namespace,
		},
		StringData: map[string]string{
			systemMessageEnvName: systemMessage,
		},
	}

	t.Logf("User configuration secret %+v", *userConfiguration)
	if err := framework.Global.Client.Create(context.TODO(), userConfiguration, nil); err != nil {
		t.Fatal(err)
	}
}

func createUserConfigurationConfigMap(t *testing.T, jenkinsCRName string, namespace string, numberOfExecutors int, systemMessage string) {
	userConfiguration := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GetUserConfigurationConfigMapName(jenkinsCRName),
			Namespace: namespace,
		},
		Data: map[string]string{
			"1-set-executors.groovy": fmt.Sprintf(`
import jenkins.model.Jenkins

Jenkins.instance.setNumExecutors(%d)
Jenkins.instance.save()`, numberOfExecutors),
			"1-casc.yaml": fmt.Sprintf(`
jenkins:
  systemMessage: "%s"`, systemMessage),
		},
	}

	t.Logf("User configuration %+v", *userConfiguration)
	if err := framework.Global.Client.Create(context.TODO(), userConfiguration, nil); err != nil {
		t.Fatal(err)
	}
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

	if !reflect.DeepEqual(jenkinsPod.Spec.NodeSelector, jenkins.Spec.Master.NodeSelector) {
		t.Fatalf("Invalid jenkins pod node selector expected '%+v', actual '%+v'", jenkins.Spec.Master.NodeSelector, jenkinsPod.Spec.NodeSelector)
	}

	requiredEnvs := resources.GetJenkinsMasterPodBaseEnvs()
	requiredEnvs = append(requiredEnvs, jenkins.Spec.Master.Env...)
	if !reflect.DeepEqual(jenkinsPod.Spec.Containers[0].Env, requiredEnvs) {
		t.Fatalf("Invalid jenkins pod continer resources expected '%+v', actual '%+v'", requiredEnvs, jenkinsPod.Spec.Containers[0].Env)
	}

	t.Log("Jenkins pod attributes are valid")
}

func verifyPlugins(t *testing.T, jenkinsClient jenkinsclient.Jenkins, jenkins *v1alpha1.Jenkins) {
	installedPlugins, err := jenkinsClient.GetPlugins(1)
	if err != nil {
		t.Fatal(err)
	}

	requiredPlugins := []map[string][]string{plugins.BasePlugins(), jenkins.Spec.Master.Plugins}
	for _, p := range requiredPlugins {
		for rootPluginName, dependentPlugins := range p {
			rootPlugin, err := plugins.New(rootPluginName)
			if err != nil {
				t.Fatal(err)
			}
			if found, ok := isPluginValid(installedPlugins, *rootPlugin); !ok {
				t.Fatalf("Invalid plugin '%s', actual '%+v'", rootPlugin, found)
			}
			for _, pluginName := range dependentPlugins {
				plugin, err := plugins.New(pluginName)
				if err != nil {
					t.Fatal(err)
				}
				if found, ok := isPluginValid(installedPlugins, *plugin); !ok {
					t.Fatalf("Invalid plugin '%s', actual '%+v'", rootPlugin, found)
				}
			}
		}
	}

	t.Log("All plugins have been installed")
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

func verifyUserConfiguration(t *testing.T, jenkinsClient jenkinsclient.Jenkins, amountOfExecutors int, systemMessage string) {
	checkConfigurationViaGroovyScript := fmt.Sprintf(`
if (!new Integer(%d).equals(Jenkins.instance.numExecutors)) {
	throw new Exception("Configuration via groovy scripts failed")
}`, amountOfExecutors)
	logs, err := jenkinsClient.ExecuteScript(checkConfigurationViaGroovyScript)
	assert.NoError(t, err, logs)

	checkConfigurationAsCode := fmt.Sprintf(`
if (!"%s".equals(Jenkins.instance.systemMessage)) {
	throw new Exception("Configuration as code failed")
}`, systemMessage)
	logs, err = jenkinsClient.ExecuteScript(checkConfigurationAsCode)
	assert.NoError(t, err, logs)
}
