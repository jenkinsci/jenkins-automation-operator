// +build !OpenShift

package e2e

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
)

func updateJenkinsCR(t *testing.T, jenkins *v1alpha2.Jenkins) {
	// do nothing
}
