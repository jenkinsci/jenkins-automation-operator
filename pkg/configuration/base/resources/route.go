package resources

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// RouteKind the kind name for route
const RouteKind = "Route"

var (
	IsRouteAPIAvailable = false
	RouteAPIChecked     = false
)

// UpdateRoute returns new route matching the service
func UpdateRoute(actual routev1.Route, jenkins *v1alpha2.Jenkins) routev1.Route {
	actualTargetService := actual.Spec.To
	serviceName := GetJenkinsHTTPServiceName(jenkins)
	if actualTargetService.Name != serviceName {
		actual.Spec.To.Name = serviceName
	}
	port := jenkins.Spec.Service.Port
	if actual.Spec.Port.TargetPort.IntVal != port {
		actual.Spec.Port.TargetPort = intstr.FromInt(int(port))
	}
	return actual
}

// IsRouteAPIAvailable tells if the Route API is installed and discoverable
/*func IsRouteAPIAvailable(client client.Client) bool {
	if routeAPIChecked {
		return isRouteAPIAvailable
	}
	gv := schema.GroupVersion{
		Group:   routev1.GroupName,
		Version: routev1.SchemeGroupVersion.Version,
	}
	if err := discovery.ServerSupportsVersion(client, gv); err != nil {
		// error, API not available
		routeAPIChecked = true
		isRouteAPIAvailable = false
	} else {
		// API Exists
		routeAPIChecked = true
		isRouteAPIAvailable = true
	}
	return isRouteAPIAvailable

}*/
