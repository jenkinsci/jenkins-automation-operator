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
	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupVolumeReconciler reconciles a BackupVolume object
type BackupVolumeReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	NotificationEvents chan event.Event
}

func (r *BackupVolumeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	backupLogger := r.Log.WithValues("backup", req.NamespacedName)

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
	if len(backupVolumeInstance.Status.Conditions) > 0 {
		return ctrl.Result{}, nil
	}
	backupLogger.Info("Jenkins Backup with name " + backupVolumeInstance.Name + " has been created")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		Complete(r)
}
