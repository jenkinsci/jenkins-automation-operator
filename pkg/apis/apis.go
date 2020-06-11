package apis

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha2.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, routev1.AddToScheme)
	AddToSchemes = append(AddToSchemes, appsv1.AddToScheme)
}
