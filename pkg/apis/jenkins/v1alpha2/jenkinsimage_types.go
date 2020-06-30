package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsImageSpec defines the desired state of JenkinsImage
type JenkinsImageSpec struct {
	From    Image           `json:"from"`
	To      Image           `json:"to"`
	Plugins []JenkinsPlugin `json:"plugins"` // Plugins list
	// DefaultUpdateCenter is a customer update center url from which all plugins will be downloaded.
	// if not specified, https://updates.jenkins.io/ is used
	DefaultUpdateCenter string `json:"defaultUpdateCenter,omitempty"`
}

// Defines Jenkins Plugin structure
type JenkinsPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	// UpdateCenter is a specific update center url from which this plugin will be downloaded. If not
	// specified, DefaultUpdateCenter is used.
	UpdateCenter string `json:"updateCenter,omitempty"`
}

// Defines Jenkins Plugin structure
type Image struct {
	Name     string `json:"name"`
	Tag      string `json:"tag,omitempty"`
	Registry string `json:"registry,omitempty"`
	// Secret is an optional reference to a secret in the same namespace to use for pushing to or pulling from the registry.
	Secret   string `json:"secret,omitempty"`
}

// JenkinsImageStatus defines the observed state of JenkinsImage
type JenkinsImageStatus struct {
	Image            string          `json:"image,omitempty"`
	MD5Sum           string          `json:"md5sum,omitempty"`
	InstalledPlugins []JenkinsPlugin `json:"installedPlugins,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsImage is the Schema for the jenkinsimages API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=jenkinsimages,scope=Namespaced
type JenkinsImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsImageSpec   `json:"spec,omitempty"`
	Status            JenkinsImageStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsImageList contains a list of JenkinsImage
type JenkinsImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsImage{}, &JenkinsImageList{})
}
