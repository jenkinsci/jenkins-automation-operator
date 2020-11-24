package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	routev1 "github.com/openshift/api/route/v1"
	stackerr "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createRoute takes the ServiceName and Creates the Route based on it
func (r *JenkinsBaseConfigurationReconciler) createRoute(meta metav1.ObjectMeta, serviceName string, jenkins *v1alpha2.Jenkins) error {
	route := routev1.Route{}
	name := fmt.Sprintf("jenkins-%s", jenkins.ObjectMeta.Name)
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &route)
	r.logger.Info(fmt.Sprintf("Got route: %s", route.Name))
	if err != nil {
		if apierrors.IsNotFound(err) {
			actual := newRoute(meta, name, serviceName)
			route = resources.UpdateRoute(actual, jenkins)
			if err = r.CreateResource(&route); err != nil {
				r.logger.Error(err, fmt.Sprintf("Error while creating (NotFound) Route: %+v : error: %+v", route, err))
				return stackerr.WithStack(err)
			}
		}
		return stackerr.WithStack(err)
	}
	route.ObjectMeta.Labels = meta.Labels // make sure that user won't break service by hand
	r.logger.Info(fmt.Sprintf("About to update route: %s", route.Name))
	route = resources.UpdateRoute(route, jenkins)
	err = r.UpdateResource(&route)
	if err != nil {
		// https://github.com/kubernetes/kubernetes/issues/28149
		if strings.Contains(err.Error(), "please apply your changes to the latest version and try again") {
			r.logger.Info(fmt.Sprintf("Route updated successfully: %s", route.Name))
		} else {
			r.logger.Error(err, fmt.Sprintf("Error while updating Route: %+v : error: %+v", route, err))
			return stackerr.WithStack(err)
		}
	}
	return nil
}

func newRouteSpect(serviceName string, port *routev1.RoutePort) routev1.RouteSpec {
	return routev1.RouteSpec{
		TLS: &routev1.TLSConfig{
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			Termination:                   routev1.TLSTerminationEdge,
		},
		To: routev1.RouteTargetReference{
			Kind: resources.ServiceKind,
			Name: serviceName,
		},
		Port: port,
	}
}

func newRoute(meta metav1.ObjectMeta, name string, serviceName string) routev1.Route {
	port := &routev1.RoutePort{
		TargetPort: intstr.FromString(""),
	}
	routeSpec := newRouteSpect(serviceName, port)
	return routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: meta.Namespace,
			Labels:    meta.Labels,
		},
		Spec: routeSpec,
	}
}
