package base

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/backuprestore"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/api/errors"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/version"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsMasterPod() (reconcile.Result, error) {
	userAndPasswordHash, err := r.calculateUserAndPasswordHash()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	now := metav1.Now()
	r.Configuration.Jenkins.Status = v1alpha2.JenkinsStatus{
		OperatorVersion:     version.Version,
		ProvisionStartTime:  &now,
		LastBackup:          r.Configuration.Jenkins.Status.LastBackup,
		PendingBackup:       r.Configuration.Jenkins.Status.LastBackup,
		UserAndPasswordHash: userAndPasswordHash,
	}

	currentJenkinsMasterPod := &corev1.Pod{}
	fmt.Printf("Labels being used %+v \n", resources.GetJenkinsMasterPodLabels(r.Jenkins))
	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(resources.GetJenkinsMasterPodLabels(r.Jenkins)).String(),
	}
	podList, err := r.ClientSet.CoreV1().Pods(r.Jenkins.Namespace).List(podListOptions)

	for {
		if podList != nil && len(podList.Items) < 1 {
			if err != nil && errors.IsNotFound(err) {
				r.logger.Info("Checking availability of Jenkins Deployment")
			} else if err == nil {
				break
			}
			podList, err = r.ClientSet.CoreV1().Pods(r.Jenkins.Namespace).List(podListOptions)
			time.Sleep(time.Millisecond * 10)
		} else if podList != nil && len(podList.Items) == 1 {
			currentJenkinsMasterPod = &podList.Items[0]
			break
		}
	}

	*r.Notifications <- event.Event{
		Jenkins: *r.Configuration.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason: reason.NewPodCreation(reason.OperatorSource,
			[]string{"Jenkins Pod created by deployment with name " + currentJenkinsMasterPod.Name}),
	}
	if err == nil {
		return reconcile.Result{Requeue: true}, r.Client.Update(context.TODO(), r.Configuration.Jenkins)
	}

	r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))

	if r.IsJenkinsTerminating(currentJenkinsMasterPod) && r.Configuration.Jenkins.Status.UserConfigurationCompletedTime != nil {
		backupAndRestore := backuprestore.New(r.Configuration, r.logger)
		if backupAndRestore.IsBackupTriggerEnabled() {
			backupAndRestore.StopBackupTrigger()
			return reconcile.Result{Requeue: true}, nil
		}
		if r.Configuration.Jenkins.Spec.Backup.MakeBackupBeforePodDeletion && !r.Configuration.Jenkins.Status.BackupDoneBeforePodDeletion {
			if r.Configuration.Jenkins.Status.LastBackup == r.Configuration.Jenkins.Status.PendingBackup {
				r.Configuration.Jenkins.Status.PendingBackup++
			}
			if err = backupAndRestore.Backup(true); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}
