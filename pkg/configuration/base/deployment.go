package base

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/version"

	stackerr "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsDeploymentIsReady() (reconcile.Result, error) {
	jenkinsDeployment, err := r.GetJenkinsDeployment()
	deploymentName := jenkinsDeployment.Name
	if err != nil {
		r.logger.Info(fmt.Sprintf("Error while getting Deployment %s: %s", deploymentName, err))
		return reconcile.Result{Requeue: true}, stackerr.WithStack(err)
	}
	if jenkinsDeployment.Status.AvailableReplicas == 0 {
		r.logger.Info(fmt.Sprintf("Deployment %s still does not have available replicas", deploymentName))
		return reconcile.Result{Requeue: true}, err
	}
	r.logger.Info(fmt.Sprintf("Deployment %s exist and has availableReplicas", deploymentName))
	if r.Jenkins.Status.Phase == constants.JenkinsStatusReinitializing {
		now := metav1.Now()
		r.Jenkins.Status.BaseConfigurationCompletedTime = &now
		r.logger.Info("Jenkins BaseConfiguration Completed after reinitialization")
	}
	r.Jenkins.Status.Phase = constants.JenkinsStatusCompleted
	err = r.Client.Update(context.TODO(), r.Configuration.Jenkins)
	if err != nil {
		r.logger.Info(fmt.Sprintf("Error while updating Jenkins %s: %s", r.Configuration.Jenkins.Name, err))
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsDeploymentIsPresent(meta metav1.ObjectMeta) (reconcile.Result, error) {
	jenkinsDeployment, err := r.GetJenkinsDeployment()
	if apierrors.IsNotFound(err) {
		jenkinsDeployment = resources.NewJenkinsDeployment(meta, r.Configuration.Jenkins)
		deploymentName := jenkinsDeployment.Name
		r.sendDeploymentCreationNotification()
		namespace := jenkinsDeployment.Namespace
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Deployment %s/%s", namespace, deploymentName))
		err := r.CreateResource(jenkinsDeployment)
		if err != nil {
			r.logger.Info(fmt.Sprintf("Error while creating Deployment %s: %s", deploymentName, err))
			return reconcile.Result{Requeue: true}, stackerr.WithStack(err)
		}
		r.logger.Info(fmt.Sprintf("Deployment %s successfully created", deploymentName))
		r.sendSuccessfulDeploymentCreationNotification(deploymentName)
		jenkinsName := r.Jenkins.Name
		creationTimestamp := jenkinsDeployment.CreationTimestamp
		r.logger.Info(fmt.Sprintf("Updating Jenkins %s to set UserAndPassword and ProvisionStartTime to %+v", jenkinsName, creationTimestamp))
		userAndPasswordHash, err := r.calculateUserAndPasswordHash()
		if err != nil {
			r.logger.Info(fmt.Sprintf("Error while calculating userAndPasswordHash for Jenkins %s: %s", jenkinsName, err))
			return reconcile.Result{}, err
		}
		r.Configuration.Jenkins.Status = v1alpha2.JenkinsStatus{
			OperatorVersion:     version.Version,
			ProvisionStartTime:  &creationTimestamp,
			LastBackup:          r.Configuration.Jenkins.Status.LastBackup,
			PendingBackup:       r.Configuration.Jenkins.Status.LastBackup,
			UserAndPasswordHash: userAndPasswordHash,
		}
		err = r.Client.Update(context.TODO(), r.Configuration.Jenkins)
		if err != nil {
			r.logger.Info(fmt.Sprintf("Error while updating Jenkins %s: %s", jenkinsName, err))
			return reconcile.Result{Requeue: true}, err
		}
	} else if err != nil {
		deploymentName := jenkinsDeployment.Name
		r.logger.Info(fmt.Sprintf("Error while getting Deployment %s and error type is different from not found: %s", deploymentName, err))
		return reconcile.Result{}, stackerr.WithStack(err)
	}
	r.logger.Info(fmt.Sprintf("Deployment %s exist or has been created without any error", jenkinsDeployment.Name))
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) sendSuccessfulDeploymentCreationNotification(deploymentName string) {
	shortMessage := fmt.Sprintf("Deployment %s successfully created", deploymentName)
	*r.Notifications <- event.Event{
		Jenkins: *r.Configuration.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewDeploymentEvent(reason.OperatorSource, []string{shortMessage}),
	}
}

func (r *ReconcileJenkinsBaseConfiguration) sendDeploymentCreationNotification() {
	*r.Notifications <- event.Event{
		Jenkins: *r.Configuration.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewDeploymentEvent(reason.OperatorSource, []string{"Creating a Jenkins Deployment"}),
	}
}
