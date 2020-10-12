package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultOpenShiftImageRegistryNamespace   = "openshift-image-registry"
	DefaultOpenShiftImageRegistryServiceName = "image-registry"
)

var (
	isImageRegistryAvailable    = false
	imageRegistryAlreadyChecked = false
)

// IsImageRegistryAvailable tells if the openshift image registry is installed and working
func IsImageRegistryAvailable(client client.Client) bool {
	if imageRegistryAlreadyChecked {
		return isImageRegistryAvailable
	}
	namespace := &corev1.Namespace{}
	clusterScope := types.NamespacedName{Name: DefaultOpenShiftImageRegistryNamespace}
	err := client.Get(context.TODO(), clusterScope, namespace)
	if errors.IsNotFound(err) {
		isImageRegistryAvailable = false
	} else {
		service := &corev1.Service{}
		inNamespace := types.NamespacedName{Name: DefaultOpenShiftImageRegistryServiceName, Namespace: DefaultOpenShiftImageRegistryNamespace}
		err := client.Get(context.TODO(), inNamespace, service)
		if errors.IsNotFound(err) { // checking that the Service exists
			isImageRegistryAvailable = false // The service does not exist means the registry is not available
		} else { // if it exists, checking that it has a cluster IP
			isImageRegistryAvailable = (len(service.Spec.ClusterIP) != 0) && service.Spec.ClusterIP != "None"
		}
	}
	// Check if cache has started
	cacheErr := (*cache.ErrCacheNotStarted)(nil)
	// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/cache/informer_cache.go#L45
	if err != nil && err.Error() != cacheErr.Error() {
		imageRegistryAlreadyChecked = true // We set this
	}
	return isImageRegistryAvailable
}
