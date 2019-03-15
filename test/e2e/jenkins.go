package e2e

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func getJenkins(t *testing.T, namespace, name string) *v1alpha1.Jenkins {
	jenkins := &v1alpha1.Jenkins{}
	namespaceName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := framework.Global.Client.Get(context.TODO(), namespaceName, jenkins); err != nil {
		t.Fatal(err)
	}

	return jenkins
}

func getJenkinsMasterPod(t *testing.T, jenkins *v1alpha1.Jenkins) *v1.Pod {
	lo := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(resources.BuildResourceLabels(jenkins)).String(),
	}
	podList, err := framework.Global.KubeClient.CoreV1().Pods(jenkins.ObjectMeta.Namespace).List(lo)
	if err != nil {
		t.Fatal(err)
	}
	if len(podList.Items) != 1 {
		t.Fatalf("Jenkins pod not found, pod list: %+v", podList)
	}
	return &podList.Items[0]
}

func createJenkinsAPIClient(jenkins *v1alpha1.Jenkins) (jenkinsclient.Jenkins, error) {
	adminSecret := &v1.Secret{}
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetOperatorCredentialsSecretName(jenkins)}
	if err := framework.Global.Client.Get(context.TODO(), namespaceName, adminSecret); err != nil {
		return nil, err
	}

	jenkinsAPIURL, err := jenkinsclient.BuildJenkinsAPIUrl(jenkins.ObjectMeta.Namespace, resources.GetJenkinsHTTPServiceName(jenkins), resources.HTTPPortInt, true, true)
	if err != nil {
		return nil, err
	}

	return jenkinsclient.New(
		jenkinsAPIURL,
		string(adminSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(adminSecret.Data[resources.OperatorCredentialsSecretTokenKey]),
	)
}

func createJenkinsCR(t *testing.T, name, namespace string, seedJob *[]v1alpha1.SeedJob) *v1alpha1.Jenkins {
	var seedJobs []v1alpha1.SeedJob
	if seedJob != nil {
		seedJobs = append(seedJobs, *seedJob...)
	}

	jenkins := &v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.JenkinsSpec{
			Master: v1alpha1.JenkinsMaster{
				Image:       "jenkins/jenkins",
				Annotations: map[string]string{"test": "label"},
				Plugins: map[string][]string{
					"audit-trail:2.4":           {},
					"simple-theme-plugin:0.5.1": {},
				},
				NodeSelector: map[string]string{"kubernetes.io/hostname": "minikube"},
				Env: []v1.EnvVar{
					{
						Name:  "TEST_ENV",
						Value: "test_env_value",
					},
				},
			},
			SeedJobs: seedJobs,
		},
	}

	t.Logf("Jenkins CR %+v", *jenkins)
	if err := framework.Global.Client.Create(context.TODO(), jenkins, nil); err != nil {
		t.Fatal(err)
	}

	return jenkins
}

func verifyJenkinsAPIConnection(t *testing.T, jenkins *v1alpha1.Jenkins) jenkinsclient.Jenkins {
	client, err := createJenkinsAPIClient(jenkins)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("I can establish connection to Jenkins API")
	return client
}

func restartJenkinsMasterPod(t *testing.T, jenkins *v1alpha1.Jenkins) {
	t.Log("Restarting Jenkins master pod")
	jenkinsPod := getJenkinsMasterPod(t, jenkins)
	err := framework.Global.Client.Delete(context.TODO(), jenkinsPod)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Jenkins master pod has been restarted")
}
