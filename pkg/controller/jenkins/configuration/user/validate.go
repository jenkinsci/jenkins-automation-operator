package user

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
)

// Validate validates Jenkins CR Spec section
func (r *ReconcileUserConfiguration) Validate(jenkins *v1alpha2.Jenkins) ([]string, error) {
	backupAndRestore := backuprestore.New(r.Client, r.ClientSet, r.logger, r.Configuration.Jenkins, r.config)
	if msg := backupAndRestore.Validate(); msg != nil {
		return msg, nil
	}

	seedJobs := seedjobs.New(r.jenkinsClient, r.Configuration, r.logger)
	return seedJobs.ValidateSeedJobs(*jenkins)
}
