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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupStrategySpec defines the desired state of BackupStrategy
type BackupStrategySpec struct {
	// QuietDownDuringBackup will put the Jenkins instance in a QuietDown mode which prevents any new builds from taking place
	QuietDownDuringBackup bool `json:"quietDownDuringBackup,omitempty"`
	// Options specifies the options provided to user to backup between. default BackupStrategy sets all to true
	Options BackupOptions `json:"backupOptions"`
	// RestartAfterRestore will restart the Jenkins instance after a Restore
	RestartAfterRestore RestartConfig `json:"restartAfterRestore"`
	// Mount Configmap containing script
	// Scheduling Backup using the BackupStrategy, will create CronJob
}

// BackupOptions specifies the options provided to user to backup between. default BackupStrategy sets all to true
type BackupOptions struct {
	Jobs    bool `json:"jobs"`
	Plugins bool `json:"plugins"`
	Config  bool `json:"config"`
}

// Config for Restart (applies only to Restore
type RestartConfig struct {
	Enabled bool `json:"enabled"`
	Safe    bool `json:"safe,omitempty"`
}

// BackupStrategyStatus defines the observed state of BackupStrategy
type BackupStrategyStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupStrategy is a reusable and referencable strategy used for backing up
// Jenkins instances and information available inside
type BackupStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupStrategySpec   `json:"spec,omitempty"`
	Status BackupStrategyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupStrategyList contains a list of BackupStrategy
type BackupStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupStrategy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupStrategy{}, &BackupStrategyList{})
}
