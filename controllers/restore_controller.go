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

	"github.com/jenkinsci/kubernetes-operator/pkg/exec"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"

	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	stackerr "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jenkinsv1alpha2 "github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
)

// RestoreReconciler reconciles a Restore object
type RestoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	restoreLogger     = log.Log.WithName("restore")
	restoreExecClient = exec.KubeExecClient{}
)

// +kubebuilder:rbac:groups=jenkins.io,resources=restores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=restores/status,verbs=get;update;patch

func (r *RestoreReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	restoreLogger := r.Log.WithValues("backup", req.NamespacedName)

	// Fetch the Restore instance
	restoreInstance := &jenkinsv1alpha2.Restore{}
	err := r.Client.Get(ctx, req.NamespacedName, restoreInstance)
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

	restoreLogger.Info("Jenkins Restore with name " + restoreInstance.Name + " has been created")

	// Fetch the Backup instance
	backupInstance := &jenkinsv1alpha2.Backup{}
	restoreNamespacedName := types.NamespacedName{
		Name:      restoreInstance.Spec.BackupRef,
		Namespace: req.Namespace,
	}
	err = r.Client.Get(ctx, restoreNamespacedName, backupInstance)
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
	backupSpec := backupInstance.Spec
	jenkinsNamespacedName := types.NamespacedName{
		Name:      backupSpec.JenkinsRef,
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
	restoreLogger.Info(fmt.Sprintf("Restore in progress for Jenkins instance '%s'", jenkinsInstance.Name))
	restoreLogger.Info(fmt.Sprintf("Jenkins '%s' for Restore '%s' found !", jenkinsInstance.Name, req.Name))

	jenkinsPod, err := r.GetPodByDeployment(jenkinsInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = restoreExecClient.InitKubeGoClient()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Running each exec in a goroutine as goclient REST SPDYRExecutor stream does not cancel
	// https://github.com/kubernetes/client-go/issues/554#issuecomment-578886198
	execComplete := make(chan bool, 1)
	execErr := make(chan error, 1)

	// Restore
	go func() {
		err = r.execRestore(restoreInstance, restoreExecClient.Client, jenkinsPod)
		execErr <- err
		execComplete <- true
	}()
	<-execComplete
	err = <-execErr
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsv1alpha2.Restore{}).
		Complete(r)
}

func (r *RestoreReconciler) execRestore(restoreInstance *jenkinsv1alpha2.Restore, clientConfig *rest.Config, jenkinsPod *corev1.Pod) error {
	execScript := []string{}
	currentBackupLocation := restoreInstance.Spec.BackupRef
	execScript = append(execScript,
		"cp -r "+resources.JenkinsBackupVolumePath+"/"+currentBackupLocation+"/*"+" "+defaultJenkinsHome)
	err := r.makeRequest(clientConfig, jenkinsPod, restoreInstance, strings.Join(execScript, " "))
	if err != nil {
		restoreLogger.Info(fmt.Sprintf("Error while Jenkins restore request %s", err.Error()))
		return err
	}
	return nil
}

func (r *RestoreReconciler) makeRequest(clientConfig *rest.Config, jenkinsPod *corev1.Pod, restoreInstance *jenkinsv1alpha2.Restore, script string) error {
	request, err := r.newScriptRequest(clientConfig, jenkinsPod, restoreInstance, script)
	if err != nil {
		return err
	}
	err = r.runPodExec(clientConfig, request, restoreInstance)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Script %s for Backup instance %s has been successful", script, restoreInstance.Spec.BackupRef))
	return nil
}

func (r *RestoreReconciler) newScriptRequest(clientConfig *rest.Config, jenkinsPod *corev1.Pod, restoreInstance *jenkinsv1alpha2.Restore, script string) (*rest.Request, error) {
	client, err := clientgocorev1.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	podExecRequest := client.RESTClient().Post().Resource("pods").
		Name(jenkinsPod.Name).
		Namespace(restoreInstance.Namespace).
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

func (r *RestoreReconciler) runPodExec(clientConfig *rest.Config, podExecRequest *rest.Request, restoreInstance *jenkinsv1alpha2.Restore) error {
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
		return err
	}
	if stdout.String() != "" {
		logger.Info(fmt.Sprintf("Backup '%s' in Namespace '%s' Execution STDOUT\n\t%s", restoreInstance.Name, restoreInstance.Namespace, stdout.String()))
	}
	if stderr.String() != "" {
		logger.Info(fmt.Sprintf("Backup '%s' in Namespace '%s' Execution STDERR\n\t%s", restoreInstance.Name, restoreInstance.Namespace, stderr.String()))
	}

	return err
}

func (r *RestoreReconciler) GetPodByDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*corev1.Pod, error) {
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
	restoreLogger.V(log.VDebug).Info(fmt.Sprintf("Successfully got the Pod: %s", pod.Name))
	return &pods.Items[0], err
}

// GetJenkinsDeployment gets the jenkins master deployment.
func (r *RestoreReconciler) GetJenkinsDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*appsv1.Deployment, error) {
	deploymentName := resources.GetJenkinsDeploymentName(jenkins)
	restoreLogger.V(log.VDebug).Info(fmt.Sprintf("Getting JenkinsDeploymentName for : %+v, querying deployment named: %s", jenkins.Name, deploymentName))
	jenkinsDeployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: deploymentName, Namespace: jenkins.Namespace}
	err := r.Client.Get(context.TODO(), namespacedName, jenkinsDeployment)
	if err != nil {
		restoreLogger.V(log.VDebug).Info(fmt.Sprintf("No deployment named: %s found: %+v", deploymentName, err))
		return nil, err
	}
	return jenkinsDeployment, nil
}

func (r *RestoreReconciler) GetReplicaSetByDeployment(jenkins *jenkinsv1alpha2.Jenkins) (*appsv1.ReplicaSet, error) {
	deployment, _ := r.GetJenkinsDeployment(jenkins)
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	replicasSetList := appsv1.ReplicaSetList{}
	if err != nil {
		restoreLogger.V(log.VDebug).Info(fmt.Sprintf("Error while getting the replicaset using selector: %s : error: %+v", selector, err))
	}
	listOptions := client.ListOptions{LabelSelector: selector}
	err = r.Client.List(context.TODO(), &replicasSetList, &listOptions)
	if err != nil || len(replicasSetList.Items) == 0 {
		restoreLogger.V(log.VDebug).Info(fmt.Sprintf("Error while getting the replicaset using selector: %s : error: %+v", selector, err))
		return nil, stackerr.Errorf("Deployment has no replicaSet attached yet: Error was: %+v", err)
	}
	replicaSet := replicasSetList.Items[0]
	restoreLogger.V(log.VDebug).Info(fmt.Sprintf("Successfully got the ReplicaSet: %s", replicaSet.Name))
	return &replicaSet, nil
}
