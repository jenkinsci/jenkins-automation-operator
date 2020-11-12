/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-lib/status"

	"github.com/jenkinsci/kubernetes-operator/pkg/exec"

	"github.com/go-logr/logr"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	stackerr "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsv1alpha2 "github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jenkins.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=backups/status,verbs=get;update;patch

var (
	logx               = log.Log
	logger             = logx.WithName("backup")
	defaultJenkinsHome = "/var/lib/jenkins"
	// BackupInitialized and other Condition Types
	BackupInitialized  status.ConditionType = "BackupInitialized"
	QuietDownStarted   status.ConditionType = "QuietDownStarted"
	BackupCompleted    status.ConditionType = "BackupCompleted"
	QuietDownCancelled status.ConditionType = "QuietDownCancelled"
)

func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsv1alpha2.Backup{}).
		Complete(r)
}

func (r *BackupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	backupLogger := r.Log.WithValues("backup", req.NamespacedName)
	execClient := exec.NewKubeExecClient()

	// Fetch the Backup instance
	backupInstance := &jenkinsv1alpha2.Backup{}
	err := r.Client.Get(ctx, req.NamespacedName, backupInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	if len(backupInstance.Status.Conditions) > 0 {
		return ctrl.Result{}, nil
	}
	backupLogger.Info("Jenkins Backup with name " + backupInstance.Name + " has been created")

	backupSpec := backupInstance.Spec
	backupConfig := &jenkinsv1alpha2.BackupConfig{}
	// Use default BackupConfig if configRef not provided
	backupConfigName := DefaultBackupConfigName
	if backupSpec.ConfigRef != "" {
		backupConfigName = backupSpec.ConfigRef
	}
	backupConfigNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      backupConfigName,
	}
	err = r.Client.Get(ctx, backupConfigNamespacedName, backupConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Fetch the Jenkins instance
	jenkinsInstance := &jenkinsv1alpha2.Jenkins{}
	jenkinsNamespacedName := types.NamespacedName{
		Name:      backupConfig.Spec.JenkinsRef,
		Namespace: req.Namespace,
	}
	err = r.Client.Get(ctx, jenkinsNamespacedName, jenkinsInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	jenkinsPod, err := r.GetPodByDeployment(jenkinsInstance)
	if err != nil {
		return ctrl.Result{}, err
	}
	backupLogger.Info(fmt.Sprintf("Jenkins '%s' for Backup '%s' found !", jenkinsInstance.Name, req.Name))

	err = execClient.InitKubeGoClient()
	if err != nil {
		backupInstance.Status.Conditions.SetCondition(status.Condition{
			Type:   BackupInitialized,
			Status: corev1.ConditionFalse,
			Reason: (status.ConditionReason)(err.Error()),
		})
		err = r.Client.Status().Update(ctx, backupInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	backupInstance.Status.Conditions.SetCondition(status.Condition{
		Type:   BackupInitialized,
		Status: corev1.ConditionTrue,
	})
	err = r.Client.Status().Update(ctx, backupInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// QuietDown
	if backupConfig.Spec.QuietDownDuringBackup {
		err := r.performJenkinsQuietDown(ctx, execClient, jenkinsPod, backupInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Backup
	err = r.performJenkinsBackup(ctx, execClient, jenkinsPod, backupInstance, backupConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	// CancelQuietDown
	if backupConfig.Spec.QuietDownDuringBackup {
		err = r.performJenkinsCancelQuietDown(ctx, execClient, jenkinsPod, backupInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	backupInstance.Status.Conditions.SetCondition(status.Condition{
		Type:   BackupCompleted,
		Status: corev1.ConditionTrue,
	})
	err = r.Client.Status().Update(ctx, backupInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BackupReconciler) performJenkinsCancelQuietDown(ctx context.Context, execClient exec.KubeExecClient, jenkinsPod *corev1.Pod, backupInstance *jenkinsv1alpha2.Backup) error {
	execCancelQuietDown := strings.Join([]string{"sh", resources.CancelQuietDownScriptPath}, " ")
	err := execClient.MakeRequest(jenkinsPod, backupInstance.Name, execCancelQuietDown)
	if err != nil {
		backupInstance.Status.Conditions.SetCondition(status.Condition{
			Type:   QuietDownCancelled,
			Status: corev1.ConditionFalse,
			Reason: (status.ConditionReason)(fmt.Sprintf("CancelQuietDown failed with error %s", err.Error())),
		})
		err = r.Client.Status().Update(ctx, backupInstance)
		if err != nil {
			return err
		}
		return nil
	}
	backupInstance.Status.Conditions.SetCondition(status.Condition{
		Type:   QuietDownCancelled,
		Status: corev1.ConditionTrue,
	})
	err = r.Client.Status().Update(ctx, backupInstance)
	if err != nil {
		return err
	}
	return nil
}

func (r *BackupReconciler) performJenkinsBackup(ctx context.Context, execClient exec.KubeExecClient, jenkinsPod *corev1.Pod, backupInstance *jenkinsv1alpha2.Backup, backupConfig *jenkinsv1alpha2.BackupConfig) error {
	backupToLocation := resources.JenkinsBackupVolumePath + "/" + backupInstance.Name + "/"
	execCreateBackupDir := strings.Join([]string{"mkdir", backupToLocation}, " ")
	err := execClient.MakeRequest(jenkinsPod, backupInstance.Name, execCreateBackupDir)
	if err != nil {
		backupInstance.Status.Conditions.SetCondition(status.Condition{
			Type:   BackupCompleted,
			Status: corev1.ConditionFalse,
			Reason: (status.ConditionReason)(fmt.Sprintf("Failed to create backup directory %s %s", backupToLocation, err.Error())),
		})
		err = r.Client.Status().Update(ctx, backupInstance)
		if err != nil {
			return err
		}
		return nil
	}
	// Create Backup based on Spec
	backupFromLocation := defaultJenkinsHome
	backupFromSubLocations := []string{}
	if backupConfig.Spec.Options.Config {
		backupFromSubLocations = append(backupFromSubLocations, "*.xml")
	}
	if backupConfig.Spec.Options.Jobs {
		backupFromSubLocations = append(backupFromSubLocations, "jobs")
	}
	if backupConfig.Spec.Options.Plugins {
		backupFromSubLocations = append(backupFromSubLocations, "plugins")
	}
	if len(backupFromSubLocations) > 0 {
		for _, sl := range backupFromSubLocations {
			// Backup each location in a different request
			backupFromSubLocation := strings.Join([]string{backupFromLocation, sl}, "/")
			execBackupSubLocation := strings.Join([]string{"cp", "-r", backupFromSubLocation, backupToLocation}, " ")
			err = execClient.MakeRequest(jenkinsPod, backupInstance.Name, execBackupSubLocation)
			if err != nil {
				backupInstance.Status.Conditions.SetCondition(status.Condition{
					Type:   BackupCompleted,
					Status: corev1.ConditionFalse,
					Reason: (status.ConditionReason)(fmt.Sprintf("Failed to backup from %s %s", backupFromSubLocation, err.Error())),
				})
				err = r.Client.Status().Update(ctx, backupInstance)
				if err != nil {
					return err
				}
				return nil
			}
		}
		err = r.Client.Status().Update(ctx, backupInstance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *BackupReconciler) performJenkinsQuietDown(ctx context.Context, execClient exec.KubeExecClient, jenkinsPod *corev1.Pod, backupInstance *jenkinsv1alpha2.Backup) error {
	execQuietDown := strings.Join([]string{"sh", resources.QuietDownScriptPath}, " ")
	err := execClient.MakeRequest(jenkinsPod, backupInstance.Name, execQuietDown)
	if err != nil {
		backupInstance.Status.Conditions.SetCondition(status.Condition{
			Type:   QuietDownStarted,
			Status: corev1.ConditionFalse,
			Reason: (status.ConditionReason)(err.Error()),
		})
		err = r.Client.Status().Update(ctx, backupInstance)
		if err != nil {
			return err
		}
		return nil
	}
	backupInstance.Status.Conditions.SetCondition(status.Condition{
		Type:   QuietDownStarted,
		Status: corev1.ConditionTrue,
	})
	err = r.Client.Status().Update(ctx, backupInstance)
	if err != nil {
		return err
	}
	return nil
}

func (r *BackupReconciler) GetPodByDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*corev1.Pod, error) {
	replicaSet, err := r.GetReplicaSetByDeployment(jenkins)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(replicaSet.Spec.Selector)
	if err != nil {
		return nil, err
	}
	listOptions := client.ListOptions{LabelSelector: selector}
	pods := corev1.PodList{}
	err = r.Client.List(context.TODO(), &pods, &listOptions)
	if err != nil || len(pods.Items) == 0 {
		return nil, stackerr.Errorf("Deployment has no pod attached yet: Error was: %+v", err)
	}
	pod := pods.Items[0]
	logger.V(log.VDebug).Info(fmt.Sprintf("Successfully got the Pod: %s", pod.Name))
	return &pods.Items[0], err
}

// GetJenkinsDeployment gets the jenkins master deployment.
func (r *BackupReconciler) GetJenkinsDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*appsv1.Deployment, error) {
	deploymentName := resources.GetJenkinsDeploymentName(jenkins)
	logger.V(log.VDebug).Info(fmt.Sprintf("Getting JenkinsDeploymentName for : %+v, querying deployment named: %s", jenkins.Name, deploymentName))
	jenkinsDeployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: deploymentName, Namespace: jenkins.Namespace}
	err := r.Client.Get(context.TODO(), namespacedName, jenkinsDeployment)
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("No deployment named: %s found: %+v", deploymentName, err))
		return nil, err
	}
	return jenkinsDeployment, nil
}

func (r *BackupReconciler) GetReplicaSetByDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*appsv1.ReplicaSet, error) {
	deployment, _ := r.GetJenkinsDeployment(jenkins)
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	replicasSetList := appsv1.ReplicaSetList{}
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while getting the replicaset using selector: %s : error: %+v", selector, err))
	}
	listOptions := client.ListOptions{LabelSelector: selector}
	err = r.Client.List(context.TODO(), &replicasSetList, &listOptions)
	if err != nil || len(replicasSetList.Items) == 0 {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while getting the replicaset using selector: %s : error: %+v", selector, err))
		return nil, stackerr.Errorf("Deployment has no replicaSet attached yet: Error was: %+v", err)
	}
	replicaSet := replicasSetList.Items[0]
	logger.V(log.VDebug).Info(fmt.Sprintf("Successfully got the ReplicaSet: %s", replicaSet.Name))
	return &replicaSet, nil
}
