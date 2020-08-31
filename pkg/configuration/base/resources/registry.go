package resources

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultOpenShiftImageRegistryNamespace   = "openshift-image-registry"
	DefaultOpenShiftImageRegistryServiceName = "image-registry"
)

var isImageRegistryAvailable = false
var imageRegistryAlreadyChecked = false

//IsImageRegistryAvailable tells if the openshift image registry is installed and working
func IsImageRegistryAvailable(clientSet *kubernetes.Clientset) bool {
	if imageRegistryAlreadyChecked {
		return isImageRegistryAvailable
	}
	_, err := clientSet.CoreV1().Namespaces().Get(DefaultOpenShiftImageRegistryNamespace, EmptyGetOptions)
	if errors.IsNotFound(err) {
		isImageRegistryAvailable = false
	} else {
		svc, err := clientSet.CoreV1().Services(DefaultOpenShiftImageRegistryNamespace).Get(DefaultOpenShiftImageRegistryServiceName, EmptyGetOptions)
		if errors.IsNotFound(err) { // checking that the Service exists
			isImageRegistryAvailable = false // The service does not exist means the registry is not available
		} else { // if it exists, checking that it has a cluster IP
			isImageRegistryAvailable = (len(svc.Spec.ClusterIP) != 0) && svc.Spec.ClusterIP != "None"
		}
	}
	imageRegistryAlreadyChecked = true // We set this
	return isImageRegistryAvailable
}
