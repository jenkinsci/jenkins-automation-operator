package base

import (
	"fmt"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/reason"
	"github.com/jenkinsci/jenkins-automation-operator/version"
	stackerr "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *JenkinsBaseConfigurationReconciler) ensureJenkinsDeploymentIsReady() (ctrl.Result, error) {
	jenkinsDeployment, err := r.GetJenkinsDeployment()
	deploymentName := jenkinsDeployment.Name
	if err != nil {
		r.logger.Info(fmt.Sprintf("Error while getting Deployment %s: %s", deploymentName, err))
		return ctrl.Result{Requeue: true}, stackerr.WithStack(err)
	}
	if jenkinsDeployment.Status.AvailableReplicas == 0 {
		r.logger.Info(fmt.Sprintf("Deployment %s still does not have available replicas", deploymentName))
		return ctrl.Result{Requeue: true}, err
	}
	r.logger.Info(fmt.Sprintf("Deployment %s exist and has availableReplicas...updating phase and completion time", deploymentName))
	r.logger.Info("Jenkins BaseConfiguration Completed after reinitialization")
	return ctrl.Result{}, nil
}

func (r *JenkinsBaseConfigurationReconciler) ensureJenkinsDeploymentIsPresent(meta metav1.ObjectMeta) (ctrl.Result, error) {
	jenkinsDeployment, err := r.GetJenkinsDeployment()
	jenkins := r.Jenkins
	namespace := jenkins.Namespace
	if err != nil {
		r.logger.Info(fmt.Sprintf("Error while getting GetJenkinsDeployment: %+v", err))
	}
	if apierrors.IsNotFound(err) {
		r.logger.Info("Error type is not found: Creating deployment")
		jenkinsDeployment = resources.NewJenkinsDeployment(meta, jenkins, jenkins.Status.Spec)
		deploymentName := jenkinsDeployment.Name
		r.logger.Info("Sending notification")
		r.sendDeploymentCreationNotification()
		r.logger.Info("Notification sent notification")
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Deployment %s/%s", namespace, deploymentName))
		err := r.CreateResource(jenkinsDeployment)
		if err != nil {
			r.logger.Info(fmt.Sprintf("Error while creating Deployment %s: %s", deploymentName, err))
			return ctrl.Result{Requeue: true}, stackerr.WithStack(err)
		}
		r.logger.Info(fmt.Sprintf("Deployment %s successfully created", deploymentName))
		// Re-read the jenkins deployment to get the update values
		jenkinsDeployment, err = r.GetJenkinsDeployment()
		if err != nil {
			r.logger.Info(fmt.Sprintf("Error while reading Deployment %s: %s", deploymentName, err))
			return ctrl.Result{Requeue: true}, stackerr.WithStack(err)
		}
		r.sendSuccessfulDeploymentCreationNotification(deploymentName)
	}

	jenkinsName := jenkins.Name
	deploymentName := jenkinsDeployment.Name
	creationTimestamp := jenkinsDeployment.CreationTimestamp
	if creationTimestamp.IsZero() {
		r.logger.Info(fmt.Sprintf("Error while getting creationTimestamp from deployment %s for Jenkins %s: %s", deploymentName, jenkinsName, err))
		return ctrl.Result{Requeue: true}, err
	}
	r.logger.Info(fmt.Sprintf("Updating Jenkins %s to set UserAndPassword and ProvisionStartTime to %+v", jenkinsName, creationTimestamp))
	r.logger.Info(fmt.Sprintf("Setting Jenkins.Status.ProvisionStartTime to deployment %s creationTimestamp: %s : %+v", deploymentName, jenkinsName, creationTimestamp))
	status := r.Jenkins.Status
	status.OperatorVersion = version.Version
	r.logger.Info(fmt.Sprintf("Deployment %s exist or has been created without any error", jenkinsDeployment.Name))
	return ctrl.Result{}, nil
}

func (r *JenkinsBaseConfigurationReconciler) sendSuccessfulDeploymentCreationNotification(deploymentName string) {
	shortMessage := fmt.Sprintf("Deployment %s successfully created", deploymentName)
	*r.Notifications <- event.Event{
		Jenkins: *r.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewDeploymentEvent(reason.OperatorSource, []string{shortMessage}),
	}
}

func (r *JenkinsBaseConfigurationReconciler) sendDeploymentCreationNotification() {
	*r.Notifications <- event.Event{
		Jenkins: *r.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason.NewDeploymentEvent(reason.OperatorSource, []string{"Creating a Jenkins Deployment"}),
	}
}
