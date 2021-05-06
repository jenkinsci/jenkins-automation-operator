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

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	BackupVolumePresent status.ConditionType = "BackupVolumePresent"
)

// BackupVolumeReconciler reconciles a BackupVolume object
type BackupVolumeReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	NotificationEvents chan event.Event
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}

func (r *BackupVolumeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	backupLogger := r.Log.WithValues("backupVolume", req.NamespacedName)

	// Fetch the Backup instance
	backupVolumeInstance := &v1alpha2.BackupVolume{}
	err := r.Client.Get(ctx, req.NamespacedName, backupVolumeInstance)
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
	backupLogger.Info("Jenkins Backup with name " + backupVolumeInstance.Name + " has been created")

	defaultStorageClassName := ""
	storageClassList := &storagev1.StorageClassList{}
	storageClassListNamespacedName := types.NamespacedName{Name: "", Namespace: req.Namespace}
	err = r.Client.Get(context.TODO(), storageClassListNamespacedName, storageClassList)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, sc := range storageClassList.Items {
		if value, ok := sc.Annotations[DefaultStorageClassLabel]; ok && value == "true" {
			defaultStorageClassName = sc.Name
		}
	}

	persistentSpec := backupVolumeInstance.Spec
	storageClassName := defaultStorageClassName
	volumeSize := "1Gi"

	if len(persistentSpec.StorageClassName) > 0 {
		storageClassName = persistentSpec.StorageClassName
	}
	if len(persistentSpec.Size) > 0 {
		volumeSize = persistentSpec.Size
	}

	backupVolumePVCName := req.Name + "-jenkins-backup"
	backupPVCNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      backupVolumePVCName,
	}

	// Fetch the Backup instance
	backupPVC := &corev1.PersistentVolumeClaim{}
	err = r.Client.Get(ctx, backupPVCNamespacedName, backupPVC)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Creating BackupVolume PVC %s in Namespace %s",
				backupPVCNamespacedName.Name,
				backupPVCNamespacedName.Namespace))
			backupPVC.Name = backupPVCNamespacedName.Name
			backupPVC.Namespace = backupPVCNamespacedName.Namespace
			backupPVC.Spec.StorageClassName = &storageClassName
			backupPVC.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			}
			backupPVC.Spec.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(volumeSize),
				},
			}
			err = r.Client.Create(context.TODO(), backupPVC)
			if err != nil {
				backupVolumeInstance.Status.Conditions.SetCondition(status.Condition{
					Type:   BackupVolumePresent,
					Status: corev1.ConditionFalse,
					Reason: (status.ConditionReason)(err.Error()),
				})
				err = r.Client.Status().Update(ctx, backupVolumeInstance)
				if err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	backupVolumeInstance.Status.Conditions.SetCondition(status.Condition{
		Type:   BackupVolumePresent,
		Status: corev1.ConditionTrue,
	})

	err = r.Client.Status().Update(ctx, backupVolumeInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
