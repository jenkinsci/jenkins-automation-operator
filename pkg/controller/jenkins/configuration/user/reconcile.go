package user

import (
	"context"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/casc"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines values required for Jenkins user configuration
type ReconcileUserConfiguration struct {
	k8sClient     k8s.Client
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
	jenkins       *v1alpha1.Jenkins
}

// New create structure which takes care of user configuration
func New(k8sClient k8s.Client, jenkinsClient jenkinsclient.Jenkins, logger logr.Logger,
	jenkins *v1alpha1.Jenkins) *ReconcileUserConfiguration {
	return &ReconcileUserConfiguration{
		k8sClient:     k8sClient,
		jenkinsClient: jenkinsClient,
		logger:        logger,
		jenkins:       jenkins,
	}
}

// Reconcile it's a main reconciliation loop for user supplied configuration
func (r *ReconcileUserConfiguration) Reconcile() (reconcile.Result, error) {
	// reconcile seed jobs
	result, err := r.ensureSeedJobs()
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.Requeue {
		return result, nil
	}

	result, err = r.ensureUserConfiguration(r.jenkinsClient)
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.Requeue {
		return result, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileUserConfiguration) ensureSeedJobs() (reconcile.Result, error) {
	seedJobs := seedjobs.New(r.jenkinsClient, r.k8sClient, r.logger)
	done, err := seedJobs.EnsureSeedJobs(r.jenkins)
	if err != nil {
		// build failed and can be recovered - retry build and requeue reconciliation loop with timeout
		if err == jobs.ErrorBuildFailed {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
		}
		// build failed and cannot be recovered
		if err == jobs.ErrorUnrecoverableBuildFailed {
			return reconcile.Result{}, nil
		}
		// unexpected error - requeue reconciliation loop
		return reconcile.Result{}, errors.WithStack(err)
	}
	// build not finished yet - requeue reconciliation loop with timeout
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileUserConfiguration) ensureUserConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	configuration := &corev1.ConfigMap{}
	namespaceName := types.NamespacedName{Namespace: r.jenkins.Namespace, Name: resources.GetUserConfigurationConfigMapNameFromJenkins(r.jenkins)}
	err := r.k8sClient.Get(context.TODO(), namespaceName, configuration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	groovyClient := groovy.New(jenkinsClient, r.k8sClient, r.logger, constants.UserConfigurationJobName, resources.JenkinsUserConfigurationVolumePath)
	err = groovyClient.ConfigureJob()
	if err != nil {
		return reconcile.Result{}, err
	}
	done, err := groovyClient.Ensure(configuration.Data, r.jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}

	configurationAsCodeClient := casc.New(jenkinsClient, r.k8sClient, r.logger, constants.UserConfigurationCASCJobName)
	err = configurationAsCodeClient.ConfigureJob()
	if err != nil {
		return reconcile.Result{}, err
	}
	done, err = configurationAsCodeClient.Ensure(r.jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}

	return reconcile.Result{}, nil
}
