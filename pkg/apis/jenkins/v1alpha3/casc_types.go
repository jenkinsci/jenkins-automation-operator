package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CascSpec defines the desired state of Casc
type CascSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Secret         SecretRef      `json:"secret"`
	Configurations []ConfigMapRef `json:"configurations"`
}

// SecretRef is reference to Kubernetes secret.
type SecretRef string

// ConfigMapRef is reference to Kubernetes ConfigMap.
type ConfigMapRef string


// CascStatus defines the observed state of Casc
type CascStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Phase is a simple, high-level summary of where the Casc is in its lifecycle.
	// There are five possible phase values:
	// Pending: The Casc has been accepted by the Kubernetes system, but one or more of the required resources have not been created.
	// Available: All of the resources for the Casc are ready.
	// Failed: At least one resource has experienced a failure.
	// Unknown: For some reason the state of the Casc phase could not be obtained.
	Phase string `json:"phase"`

	LastTransitionTime metav1.Time          `json:"lastTransitionTime"`
	Reason             string               `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	Message            string               `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Casc is the Schema for the cascs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cascs,scope=Namespaced
type Casc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CascSpec   `json:"spec,omitempty"`
	Status CascStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CascList contains a list of Casc
type CascList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Casc `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Casc{}, &CascList{})
}
