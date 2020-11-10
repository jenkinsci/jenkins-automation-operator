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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/exec"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	stackerr "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	backupExecClient   = exec.KubeExecClient{}
)

func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsv1alpha2.Backup{}).
		Complete(r)
}

func (r *BackupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	backupLogger := r.Log.WithValues("backup", req.NamespacedName)

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
	backupLogger.Info(fmt.Sprintf("Backup in progress for Jenkins instance '%s'", jenkinsInstance.Name))
	backupLogger.Info(fmt.Sprintf("Jenkins '%s' for Backup '%s' found !", jenkinsInstance.Name, req.Name))

	err = backupExecClient.InitKubeGoClient()
	if err != nil {
		return ctrl.Result{}, err
	}

	jenkinsPod, err := r.GetPodByDeployment(jenkinsInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Running each exec in a goroutine as goclient REST SPDYRExecutor stream does not cancel
	// https://github.com/kubernetes/client-go/issues/554#issuecomment-578886198
	execComplete := make(chan bool, 1)
	execErr := make(chan error, 1)

	// QuietDown
	if backupConfig.Spec.QuietDownDuringBackup {
		go func() {
			err = r.execQuietDown(req.Name, backupExecClient.Client, jenkinsPod)
			execErr <- err
			execComplete <- true
		}()
		<-execComplete
		err = <-execErr
		if err != nil {
			logger.Info(fmt.Sprintf("Backup failed with error %s", err.Error()))
			return ctrl.Result{}, nil
		}
	}

	// Backup
	go func() {
		err = r.execBackup(backupInstance, backupConfig, backupExecClient.Client, jenkinsPod)
		execErr <- err
		execComplete <- true
	}()
	<-execComplete
	err = <-execErr
	if err != nil {
		logger.Info(fmt.Sprintf("Backup failed with error %s", err.Error()))
		return ctrl.Result{}, nil
	}

	// CancelQuietDown
	if backupConfig.Spec.QuietDownDuringBackup {
		go func() {
			err = r.execCancelQuietDown(req.Name, backupExecClient.Client, jenkinsPod)
			execErr <- err
			execComplete <- true
		}()
		<-execComplete
		err = <-execErr
		if err != nil {
			logger.Info(fmt.Sprintf("Backup failed with error %s", err.Error()))
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *BackupReconciler) execQuietDown(backupName string, clientConfig *rest.Config, jenkinsPod *corev1.Pod) error {
	execScript := []string{}
	execScript = append(execScript, "sh", resources.QuietDownScriptPath)
	err := r.makeRequest(clientConfig, jenkinsPod, backupName, strings.Join(execScript, " "))
	if err != nil {
		logger.Info(fmt.Sprintf("Error while Jenkins quietDown request %s", err.Error()))
		return err
	}
	return nil
}

func (r *BackupReconciler) execCancelQuietDown(backupName string, clientConfig *rest.Config, jenkinsPod *corev1.Pod) error {
	execScript := []string{}
	execScript = append(execScript, "sh", resources.CancelQuietDownScriptPath)
	err := r.makeRequest(clientConfig, jenkinsPod, backupName, strings.Join(execScript, " "))
	if err != nil {
		logger.Info(fmt.Sprintf("Error while Jenkins cancelQuietDown request %s", err.Error()))
		return err
	}
	return nil
}

func (r *BackupReconciler) execBackup(backupInstance *jenkinsv1alpha2.Backup, backupConfig *jenkinsv1alpha2.BackupConfig, clientConfig *rest.Config, jenkinsPod *corev1.Pod) error {
	execScript := []string{}
	currentBackupLocation := backupInstance.Name
	toLocation := resources.JenkinsBackupVolumePath + "/" + currentBackupLocation + "/"
	restoreLogger.Info("Using BackupConfig " + backupConfig.Name)
	fromLocation := defaultJenkinsHome
	// fromLocations := []string{}
	//
	// if backupConfig.Spec.Options.Config {
	//	fromLocations = append(fromLocations, "*.xml")
	// }
	// if backupConfig.Spec.Options.Jobs {
	//	fromLocations = append(fromLocations, "jobs")
	// }
	// if backupConfig.Spec.Options.Plugins {
	//	fromLocations = append(fromLocations, "plugins")
	// }
	//
	// fromLocation += "/{" + strings.Join(fromLocations, ",") + "}"
	execScript = append(execScript,
		"cp", "-r", fromLocation, toLocation)
	err := r.makeRequest(clientConfig, jenkinsPod, backupInstance.Name, strings.Join(execScript, " "))
	if err != nil {
		restoreLogger.Info(fmt.Sprintf("Error while Jenkins backup request %s", err.Error()))
		return err
	}
	return nil
}

func (r *BackupReconciler) makeRequest(clientConfig *rest.Config, jenkinsPod *corev1.Pod, backupName, script string) error {
	request, err := r.newScriptRequest(clientConfig, jenkinsPod, script)
	if err != nil {
		return err
	}
	err = r.runPodExec(clientConfig, request, backupName)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Script %s for Jenkins instance %s has been successful", script, backupName))
	return nil
}

func (r *BackupReconciler) newScriptRequest(clientConfig *rest.Config, jenkinsPod *corev1.Pod, script string) (*rest.Request, error) {
	client, err := clientgocorev1.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	podExecRequest := client.RESTClient().Post().Resource("pods").
		Name(jenkinsPod.Name).
		Namespace(jenkinsPod.Namespace).
		SubResource("exec")
	podExecOptions := &corev1.PodExecOptions{
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
		Container: "backup",
		Command: []string{
			"sh", "-c", script,
		},
	}
	logger.Info(strings.Join([]string{
		"sh", "-c", script,
	}, " "))

	podExecRequest.VersionedParams(podExecOptions, scheme.ParameterCodec)
	return podExecRequest, err
}

func (r *BackupReconciler) runPodExec(clientConfig *rest.Config, podExecRequest *rest.Request, backupName string) error {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	remoteCommand, err := remotecommand.NewSPDYExecutor(clientConfig, "POST", podExecRequest.URL())
	if err != nil {
		logger.Info(fmt.Sprintf("Error while crerating remote executor for backup %s", err.Error()))
		return err
	}

	streamOptions := remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	}
	err = remoteCommand.Stream(streamOptions)
	if err != nil {
		logger.Info(fmt.Sprintf("Error while executing backup %s", err.Error()))
		logger.Info(fmt.Sprintf("Backup '%s' Execution STDERR\n\t%s", backupName, stderr.String()))
		return err
	}
	logger.Info(fmt.Sprintf("Backup '%s' Execution STDOUT\n\t%s", backupName, stdout.String()))

	return err
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
