package user

import (
	"github.com/go-logr/logr"
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines API client for reconcile User Configuration
type ReconcileUserConfiguration interface {
	ReconcileOthers() (reconcile.Result, error)
	Validate(jenkins *v1alpha2.Jenkins) ([]string, error)
}

type reconcileUserConfiguration struct {
	configuration.Configuration
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
}

// New create structure which takes care of user configuration.
func New(configuration configuration.Configuration, jenkinsClient jenkinsclient.Jenkins) ReconcileUserConfiguration {
	return &reconcileUserConfiguration{
		Configuration: configuration,
		jenkinsClient: jenkinsClient,
		logger:        log.Log.WithValues("cr", configuration.Jenkins.Name),
	}
}

// Reconcile it's a main reconciliation loop for user supplied configuration
func (r *reconcileUserConfiguration) ReconcileOthers() (reconcile.Result, error) {
	backupAndRestore := backuprestore.New(r.Configuration, r.logger)

	if err := backupAndRestore.Restore(r.jenkinsClient); err != nil {
		return reconcile.Result{}, err
	}

	if err := backupAndRestore.Backup(false); err != nil {
		return reconcile.Result{}, err
	}
	if err := backupAndRestore.EnsureBackupTrigger(); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
