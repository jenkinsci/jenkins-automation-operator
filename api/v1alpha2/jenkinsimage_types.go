package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// A JenkinsImage definition
type Image struct {
	// The Image name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`
	// The tag name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Tag string `json:"tag,omitempty"`
	// The registry to pull from or push to this image in the form fully.qdn/myrepository/
	// Image name will be appended for push
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Registry string `json:"registry,omitempty"`
	// Secret is an optional reference to a secret in the same namespace to use for pushing to or pulling from the registry.
	// +operator-sdk:csv:customresourcedefinitions:xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Secret string `json:"secret,omitempty"`
}

type JenkinsImageBuild struct {
	// +operator-sdk:csv:customresourcedefinitions.type=status
	Image            string `json:"image,omitempty"`
	MD5Sum           string `json:"md5sum,omitempty"`
	InstalledPlugins string `json:"installedPlugins,omitempty"`
}

const (
	ImageBuildSuccessful JenkinsImagePhase = "ImageBuildSuccessful"
	ImageBuildPending    JenkinsImagePhase = "ImageBuildPending"
)

type JenkinsImagePhase string

// JenkinsImageStatus defines the observed state of JenkinsImage
type JenkinsImageStatus struct {
	Phase  JenkinsImagePhase   `json:"phase,default:ImageBuildPending"`
	Builds []JenkinsImageBuild `json:"builds"`
}

// +kubebuilder:object:root=true

// JenkinsImage is the Schema for the jenkinsimages API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=jenkinsimages,scope=Namespaced
type JenkinsImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsImageSpec   `json:"spec,omitempty"`
	Status            JenkinsImageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// JenkinsImageList contains a list of JenkinsImage
type JenkinsImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsImage{}, &JenkinsImageList{})
}
