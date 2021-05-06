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

// BackupVolumeSpec defines the desired state of BackupVolume
type BackupVolumeSpec struct {
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName,omitempty"`
	StorageClassName          string `json:"storageClassName,omitempty"`
	Size                      string `json:"size,omitempty"`
}

// BackupVolumeStatus defines the observed state of BackupVolume
type BackupVolumeStatus struct {
	Conditions status.Conditions `json:"conditions,omitempty"`
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
