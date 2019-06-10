package user

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
)

// Validate validates Jenkins CR Spec section
func (r *ReconcileUserConfiguration) Validate(jenkins *v1alpha2.Jenkins) (bool, error) {
	seedJobs := seedjobs.New(r.jenkinsClient, r.k8sClient, r.logger)
	return seedJobs.ValidateSeedJobs(*jenkins)
}
