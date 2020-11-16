package resources

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	DefaultOpenShiftImageRegistryNamespace   = "openshift-image-registry"
	DefaultOpenShiftImageRegistryServiceName = "image-registry"
)

var (
	isImageRegistryAvailable    = false
	imageRegistryAlreadyChecked = false
)

func IsImageRegistryAvailableCached() bool {
	return isImageRegistryAvailable
}

// IsImageRegistryAvailable tells if the openshift image registry is installed and working
func IsImageRegistryAvailable(manager manager.Manager) bool {
	logger := log.WithName("imageregistry_check")
	if imageRegistryAlreadyChecked {
		logger.Info(fmt.Sprintf("Already checked: return isImageRegistryAvailable %+v", isImageRegistryAvailable))
		return isImageRegistryAvailable
	}
	namespace := &corev1.Namespace{}
	clusterScope := types.NamespacedName{Name: DefaultOpenShiftImageRegistryNamespace}
	client := manager.GetAPIReader()
	err := client.Get(context.TODO(), clusterScope, namespace)
	if errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Namespace %s not found", DefaultOpenShiftImageRegistryNamespace))
		isImageRegistryAvailable = false
	} else {
		logger.Info(fmt.Sprintf("Namespace %s found", namespace.Name))
		service := &corev1.Service{}
		inNamespace := types.NamespacedName{Name: "image-registry", Namespace: namespace.Name}
		err := client.Get(context.TODO(), inNamespace, service)
		if errors.IsNotFound(err) { // checking that the Service exists
			logger.Info(fmt.Sprintf("Service %s/%s not found", DefaultOpenShiftImageRegistryNamespace, DefaultOpenShiftImageRegistryServiceName))
			isImageRegistryAvailable = false // The service does not exist means the registry is not available
		} else { // if it exists, checking that it has a cluster IP
			isImageRegistryAvailable = (len(service.Spec.ClusterIP) != 0) && service.Spec.ClusterIP != "None"
			imageRegistryAlreadyChecked = false // We set this
			logger.Info(fmt.Sprintf("Service %s/%s has a cluster IP Service: (%+v)? %+v", DefaultOpenShiftImageRegistryNamespace, DefaultOpenShiftImageRegistryServiceName, service.Spec.ClusterIP, isImageRegistryAvailable))
		}
	}
	// Check if cache has started
	cacheErr := (*cache.ErrCacheNotStarted)(nil)
	// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/cache/informer_cache.go#L45
	if err != nil && err.Error() != cacheErr.Error() {
		logger.Info("Cache not started correctly: resetting imageRegistryAlreadyChecked set to false")
		imageRegistryAlreadyChecked = false // We set this
	}
	return isImageRegistryAvailable
}
