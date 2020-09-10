package e2e

import (
	goctx "context"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jenkinsci/kubernetes-operator/internal/try"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	retryInterval               = time.Second * 5
	timeout                     = time.Second * 60
	namespaceTerminationTimeout = time.Second * 120
	retries                     = 30
)

// checkConditionFunc is used to check if a condition for the jenkins CR is set
type checkConditionFunc func(*v1alpha2.Jenkins, error) bool
type checkCascConditionFunc func(*v1alpha3.Casc, error) bool

func waitForJobToFinish(t *testing.T, job *gojenkins.Job, tick, timeout time.Duration) {
	err := try.Until(func() (end bool, err error) {
		t.Logf("Waiting for job `%s` to finish", job.GetName())
		running, err := job.IsRunning()
		if err != nil {
			return false, err
		}
		queued, err := job.IsQueued()
		if err != nil {
			return false, err
		}
		if !running && !queued {
			return true, nil
		}
		return false, nil
	}, tick, timeout)
	require.NoError(t, err)
}

func waitForJenkinsBaseConfigurationToComplete(t *testing.T, jenkins *v1alpha2.Jenkins) {
	t.Logf("Waiting for Jenkins base configuration to complete: Will retry %+v times every %+v", retries, retryInterval)
	_, err := WaitUntilJenkinsConditionSet(retryInterval, retries, jenkins, func(jenkins *v1alpha2.Jenkins, err error) bool {
		completedTime := jenkins.Status.BaseConfigurationCompletedTime
		t.Logf("Current Jenkins status BaseConfigurationCompletedTime: '%+v', error '%s'", completedTime, err)
		return err == nil && completedTime != nil
	})
	if err != nil {
		t.Errorf("Waiting for BaseConfiguration to complete failed with : %+v", jenkins)
		t.Fatal(err)
	}
	t.Log("Jenkins pod is running")
	// update jenkins CR because Operator sets default values
	namespacedName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
	err = framework.Global.Client.Get(goctx.TODO(), namespacedName, jenkins)
	assert.NoError(t, err)
}

func waitForRecreateJenkinsMasterPod(t *testing.T, jenkins *v1alpha2.Jenkins) {
	t.Logf("Waiting for Jenkins Master Pod to be ready: Will every %+v until %v", retryInterval, 30*retryInterval)
	IsJenkinsMasterPodRecreated := func() (bool, error) {
		lo := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(resources.GetJenkinsMasterPodLabels(jenkins)).String(),
		}
		podList, err := framework.Global.KubeClient.CoreV1().Pods(jenkins.ObjectMeta.Namespace).List(lo)
		if err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}
		return podList.Items[0].DeletionTimestamp == nil, nil
	}
	err := wait.Poll(retryInterval, 30*retryInterval, IsJenkinsMasterPodRecreated)
	if err != nil {
		t.Errorf("Waiting for Jenkins Master Pod to recreate failed with jenkins: %+v", jenkins)
		t.Fatal(err)
	}
	t.Log("Jenkins pod has been recreated")
}

func waitForJenkinsUserConfigurationToComplete(t *testing.T, casc *v1alpha3.Casc) {
	t.Log("Waiting for Jenkins user configuration to complete")
	_, err := WaitUntilCascConditionSet(retryInterval, retries, casc, func(casc *v1alpha3.Casc, err error) bool {
		phase := casc.Status.Phase
		t.Logf("Current Casc status phase: '%+v', error '%s'", phase, err)
		return err == nil && phase == constants.JenkinsStatusCompleted
	})
	if err != nil {
		t.Errorf("Waiting for Jenkins user configuration to complete failed with casc: %+v", casc)
		t.Fatal(err)
	}
	t.Logf("Casc completed with casc: %+v", casc)
}

func waitForJenkinsSafeRestart(t *testing.T, jenkinsClient jenkinsclient.Jenkins) {
	err := try.Until(func() (end bool, err error) {
		status, err := jenkinsClient.Poll()
		if err != nil {
			return false, err
		}
		if status != http.StatusOK {
			return false, errors.Wrap(err, "couldn't poll data from Jenkins API")
		}
		return true, nil
	}, time.Second, time.Second*70)
	require.NoError(t, err)
}

// WaitUntilCascConditionSet retries until the specified condition check becomes true for the casc CR
func WaitUntilCascConditionSet(retryInterval time.Duration, retries int, casc *v1alpha3.Casc, checkCondition checkCascConditionFunc) (*v1alpha3.Casc, error) {
	cascStatus := &v1alpha3.Casc{}
	err := wait.Poll(retryInterval, time.Duration(retries)*retryInterval, func() (bool, error) {
		namespacedName := types.NamespacedName{Namespace: casc.Namespace, Name: casc.Name}
		err := framework.Global.Client.Get(goctx.TODO(), namespacedName, cascStatus)
		return checkCondition(cascStatus, err), nil
	})
	if err != nil {
		return nil, err
	}
	return cascStatus, nil
}

// WaitUntilJenkinsConditionSet retries until the specified condition check becomes true for the jenkins CR
func WaitUntilJenkinsConditionSet(retryInterval time.Duration, retries int, jenkins *v1alpha2.Jenkins, checkCondition checkConditionFunc) (*v1alpha2.Jenkins, error) {
	jenkinsStatus := &v1alpha2.Jenkins{}
	err := wait.Poll(retryInterval, time.Duration(retries)*retryInterval, func() (bool, error) {
		namespacedName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
		err := framework.Global.Client.Get(goctx.TODO(), namespacedName, jenkinsStatus)
		return checkCondition(jenkinsStatus, err), nil
	})
	if err != nil {
		return nil, err
	}
	return jenkinsStatus, nil
}

func waitUntilNamespaceDestroyed(namespace string) error {
	isNamespaceDestroyed := func() (bool, error) {
		var namespaceList v1.NamespaceList
		err := framework.Global.Client.List(context.TODO(), &namespaceList, &client.ListOptions{})
		if err != nil {
			return true, err
		}
		exists := false
		for _, namespaceItem := range namespaceList.Items {
			if namespaceItem.Name == namespace {
				exists = true
				break
			}
		}
		return !exists, nil
	}
	err := try.Until(isNamespaceDestroyed, time.Second, namespaceTerminationTimeout)
	return err
}
