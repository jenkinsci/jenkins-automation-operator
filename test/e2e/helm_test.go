// +build Helm

package e2e

import (
	"fmt"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base"
	"os/exec"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLintHelmChart(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("helm", "lint", "./chart/jenkins-operator")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestDeployHelmChart(t *testing.T) {
	// Given
	t.Parallel()
	ctx := framework.NewContext(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err)

	jenkinsServiceList := &v1alpha2.JenkinsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.Kind,
			APIVersion: v1alpha2.SchemeGroupVersion.String(),
		},
	}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, jenkinsServiceList)
	require.NoError(t, err)

	cascServiceList := &v1alpha3.CascList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha3.Kind,
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
		},
	}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, cascServiceList)
	require.NoError(t, err)

	annotations := map[string]string{base.UseDeploymentAnnotation: "false"}
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:        "jenkins",
			Namespace:   namespace,
			Annotations: annotations,
		},
	}
	jenkins.Annotations[base.UseDeploymentAnnotation] = "false"

	casc := &v1alpha3.Casc{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "casc-example",
			Namespace: namespace,
		},
	}

	cmd := exec.Command("helm", "upgrade", "jenkins", "./chart/jenkins-operator", "--namespace", namespace, "--debug",
		"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace), "--install")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, casc)

	cmd = exec.Command("helm", "upgrade", "jenkins", "./chart/jenkins-operator", "--namespace", namespace, "--debug",
		"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace), "--install")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	// Then
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, casc)
}
