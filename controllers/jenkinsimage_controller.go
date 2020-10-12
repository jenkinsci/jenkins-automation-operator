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
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	jenkinsv1alpha2 "github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const YamlMultilineDataFieldCutSet = "|\n "

// JenkinsImageReconciler reconciles a JenkinsImage object
type JenkinsImageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *JenkinsImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsv1alpha2.JenkinsImage{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=jenkins.io,resources=jenkinsimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=jenkinsimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;create;update;patch;delete

func (r *JenkinsImageReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	reqLogger := r.Log.WithValues("jenkinsimage", request.NamespacedName)

	// Fetch the JenkinsImage instance
	instance := &jenkinsv1alpha2.JenkinsImage{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
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

	// Define a new ConfigMap containing the Dockerfile used to build the image
	dockerfile := resources.NewDockerfileConfigMap(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dockerfile, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: dockerfile.Name, Namespace: dockerfile.Namespace}, foundConfigMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", dockerfile.Namespace, "ConfigMap.Name", dockerfile.Name)
			err = r.Client.Create(context.TODO(), dockerfile)
			if err != nil {
				return ctrl.Result{}, err
			}
			// ConfigMap created successfully - don't requeue
			return ctrl.Result{}, nil
		}
	}
	// ConfigMap already exists - don't requeue
	reqLogger.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)

	// Define a new Pod object
	pod := resources.NewBuilderPod(r, instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	// Check if this Pod already exists
	foundPod := &corev1.Pod{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, foundPod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			err = r.Client.Create(context.TODO(), pod)
			if err != nil {
				return ctrl.Result{}, err
			}
			// Pod created successfully - don't requeue
			return ctrl.Result{}, nil
		}
	}
	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", foundPod.Namespace, "Pod.Name", foundPod.Name)
	go r.updateJenkinsImageStatusWhenPodIsCompleted(ctx, foundPod, instance)

	return ctrl.Result{}, nil
}

func (r *JenkinsImageReconciler) updateJenkinsImageStatusWhenPodIsCompleted(ctx context.Context, pod *corev1.Pod, instance *jenkinsv1alpha2.JenkinsImage) {
	err := resources.WaitForPodIsCompleted(r.Client, pod.Name, pod.Namespace)
	if err == nil && len(pod.Status.ContainerStatuses) == 1 {
		status := pod.Status.ContainerStatuses[0]
		if status.State.Terminated != nil {
			builtImage := strings.Trim(status.State.Terminated.Message, YamlMultilineDataFieldCutSet)
			r.Log.Info(fmt.Sprintf("Found built image (trimed): %s", builtImage))
			dockerfileContent, _ := r.getDockerfileContent(instance)
			dockerfileMD5 := strings.Trim(fmt.Sprintf("%x", md5.Sum([]byte(dockerfileContent))), YamlMultilineDataFieldCutSet)
			r.Log.Info(fmt.Sprintf("Found image checksum (trimed): %s", dockerfileMD5))
			build := jenkinsv1alpha2.JenkinsImageBuild{
				MD5Sum: dockerfileMD5,
				Image:  builtImage,
			}
			if !contains(instance.Status.Builds, build) {
				instance.Status.Builds = append(instance.Status.Builds, build)
				r.Log.Info("Updating JenkinsImage with containerStatus from Pod")
				instance.Status.Phase = jenkinsv1alpha2.ImageBuildSuccessful
				err = r.Status().Update(ctx, instance)
				if err != nil {
					// FIXME We may need go routine error handling using a dedicated channel here
					r.Log.Error(err, "Failed to update JenkinsImage status")
				}
			}
		}
	} else {
		r.Log.Info(fmt.Sprintf("Error while waiting for pod to complete: %+v, containerStatus: %+v", err, pod.Status.ContainerStatuses))
	}
}

func (r *JenkinsImageReconciler) getDockerfileContent(instance *jenkinsv1alpha2.JenkinsImage) (string, error) {
	configMapName := resources.GetDockerfileConfigMapName(instance)
	configMap := &corev1.ConfigMap{}
	// Fetch the ConfigMap instance
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Name, Name: configMapName}, configMap)
	if err != nil {
		return "", err
	}
	imageSHA256 := strings.Trim(configMap.Data[resources.DockerfileName], YamlMultilineDataFieldCutSet)
	return imageSHA256, nil
}

func contains(s []jenkinsv1alpha2.JenkinsImageBuild, e jenkinsv1alpha2.JenkinsImageBuild) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
