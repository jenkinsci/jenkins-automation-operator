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

package v1alpha2

import (
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupVolumeSpec defines the desired state of BackupVolume
type BackupVolumeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BackupVolume. Edit backupvolume_types.go to remove/update
	StorageClassName string `json:"storageClassName,omitempty"`
	Size             string `json:"size,omitempty"`
}

// BackupVolumeStatus defines the observed state of BackupVolume
type BackupVolumeStatus struct {
	Conditions                status.Conditions `json:"conditions,omitempty"`
	PersistentVolumeClaimName string            `json:"persistentVolumeClaimName,omitempty"`
	BoundPersistentVolumeName string            `json:"boundPersistentVolumeName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BackupVolume is the Schema for the backupvolumes API
type BackupVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupVolumeSpec   `json:"spec,omitempty"`
	Status BackupVolumeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupVolumeList contains a list of BackupVolume
type BackupVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupVolume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupVolume{}, &BackupVolumeList{})
}
