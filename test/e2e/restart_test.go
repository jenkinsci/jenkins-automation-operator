package e2e

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJenkinsPodRestart(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)

	defer showLogsAndCleanup(t, ctx)

	jenkinsCRName := "e2e"
	jenkins := createSimpleJenkinsCR(t, jenkinsCRName, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	deleteJenkinsPod(t, jenkins)
	waitForJenkinsPodRecreation(t, jenkins)
	checkBaseConfigurationCompleteTimeIsNotSet(t, jenkins)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
}

func TestSafeRestart(t *testing.T) {
	if skipTestSafeRestart {
		t.Skip()
	}
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkinsCRName := e2e
	cascCRName := cascE2e
	configureAuthorizationToUnSecure(t, namespace, userConfigurationConfigMapName)
	groovyScripts := v1alpha3.Customization{
		Configurations: []v1alpha3.ConfigMapRef{
			{
				Name: userConfigurationConfigMapName,
			},
		},
	}
	jenkins := createJenkinsCR(t, jenkinsCRName, namespace, "", nil)
	casc := createCascCR(t, jenkinsCRName, cascCRName, namespace, groovyScripts, v1alpha3.Customization{})
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, casc)
	jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc()
	checkIfAuthorizationStrategyUnsecuredIsSet(t, jenkinsClient)

	err := jenkinsClient.SafeRestart()
	require.NoError(t, err)
	waitForJenkinsSafeRestart(t, jenkinsClient)

	checkIfAuthorizationStrategyUnsecuredIsSet(t, jenkinsClient)
}

func configureAuthorizationToUnSecure(t *testing.T, namespace, configMapName string) {
	limitRange := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"set-unsecured-authorization.groovy": `
import hudson.security.*

def jenkins = jenkins.model.Jenkins.getInstance()

def strategy = new AuthorizationStrategy.Unsecured()
jenkins.setAuthorizationStrategy(strategy)
jenkins.save()
`,
		},
	}

	err := framework.Global.Client.Create(context.TODO(), limitRange, nil)
	require.NoError(t, err)
}

func checkIfAuthorizationStrategyUnsecuredIsSet(t *testing.T, jenkinsClient jenkinsclient.Jenkins) {
	logs, err := jenkinsClient.ExecuteScript(`
	import hudson.security.*
	  
	def jenkins = jenkins.model.Jenkins.getInstance()
	
	if (!(jenkins.getAuthorizationStrategy() instanceof AuthorizationStrategy.Unsecured)) {
	  throw new Exception('AuthorizationStrategy.Unsecured is not set')
	}
	`)
	require.NoError(t, err, logs)
}

func checkBaseConfigurationCompleteTimeIsNotSet(t *testing.T, jenkins *v1alpha2.Jenkins) {
	jenkinsStatus := &v1alpha2.Jenkins{}
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
	err := framework.Global.Client.Get(context.TODO(), namespaceName, jenkinsStatus)
	if err != nil {
		t.Fatal(err)
	}
	if jenkinsStatus.Status.BaseConfigurationCompletedTime != nil {
		t.Fatalf("Status.BaseConfigurationCompletedTime is set after pod restart, status %+v", jenkinsStatus.Status)
	}
}
