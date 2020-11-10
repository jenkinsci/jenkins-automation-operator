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

// BackupConfigSpec defines the desired state of BackupConfig
type BackupConfigSpec struct {
	// JenkinsRef points to the Jenkins restore on which Backup or Restore will be performed
	JenkinsRef string `json:"jenkinsRef"`
	// QuietDownDuringBackup will put the Jenkins instance in a QuietDown mode which prevents any new builds from taking place
	QuietDownDuringBackup bool `json:"quietDownDuringBackup,omitempty"`
	// Options specifies the options provided to user to backup between. default BackupConfig sets all to true
	Options BackupOptions `json:"backupOptions"`
	// RestartAfterRestore will restart the Jenkins instance after a Restore
	RestartAfterRestore RestartConfig `json:"restartAfterRestore"`
	// Mount Configmap containing script
	// Scheduling Backup using the BackupConfig, will create CronJob
}

// BackupOptions specifies the options provided to user to backup between. default BackupConfig sets all to true
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

// BackupConfigStatus defines the observed state of BackupConfig
type BackupConfigStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupConfig is the Schema for the backupconfigs API
type BackupConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupConfigSpec   `json:"spec,omitempty"`
	Status BackupConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupConfigList contains a list of BackupConfig
type BackupConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupConfig{}, &BackupConfigList{})
}
