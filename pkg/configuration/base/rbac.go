package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	logx "github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jenkinsci/jenkins-automation-operator/pkg/configuration/base/resources"
	stackerr "github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = logx.Log
)

const (
	EditClusterRole       = "edit"
	AuthorizationAPIGroup = "rbac.authorization.k8s.io"
)

func (r *JenkinsBaseConfigurationReconciler) createRBAC(jenkins *v1alpha2.Jenkins) error {
	meta := resources.NewResourceObjectMeta(jenkins)
	err := r.createServiceAccount(jenkins)
	if err != nil {
		return err
	}

	role := resources.NewRole(jenkins)
	err = r.CreateOrUpdateResource(role)
	if err != nil {
		return stackerr.WithStack(err)
	}

	jenkinsRole := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     meta.Name,
	}
	roleBinding := resources.NewRoleBinding(jenkins, jenkinsRole)
	roleBinding.Name = meta.Name
	err = r.CreateOrUpdateResource(roleBinding)
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func (r *JenkinsBaseConfigurationReconciler) ensureExtraRBACArePresent() error {
	roles := r.Configuration.Jenkins.Status.Spec.Roles
	jenkins := r.Jenkins
	logger.Info(fmt.Sprintf("jenkins.Roles is %+v", roles))
	for _, roleRef := range roles {
		roleBinding := resources.NewRoleBinding(jenkins, roleRef)
		err := r.CreateOrUpdateResource(roleBinding)
		if err != nil {
			return stackerr.WithStack(err)
		}
	}
	roleBindings := &rbacv1.RoleBindingList{}
	err := r.Client.List(context.TODO(), roleBindings, client.InNamespace(r.Configuration.Jenkins.Namespace))
	if err != nil {
		return stackerr.WithStack(err)
	}
	for _, roleBinding := range roleBindings.Items {
		if !strings.HasPrefix(roleBinding.Name, resources.GetExtraRoleBindingName(jenkins, rbacv1.RoleRef{Kind: "Role"})) &&
			!strings.HasPrefix(roleBinding.Name, resources.GetExtraRoleBindingName(jenkins, rbacv1.RoleRef{Kind: "ClusterRole"})) {
			continue
		}

		found := false
		for _, roleRef := range roles {
			name := resources.GetExtraRoleBindingName(jenkins, roleRef)
			if roleBinding.Name == name {
				found = true

				continue
			}
		}
		if !found {
			r.logger.Info(fmt.Sprintf("Deleting RoleBinding '%s'", roleBinding.Name))
			if err = r.Client.Delete(context.TODO(), &roleBinding); err != nil {
				return stackerr.WithStack(err)
			}
		}
	}
	return nil
}

func (r *JenkinsBaseConfigurationReconciler) GetDefaultRoleBinding(jenkins *v1alpha2.Jenkins) *rbacv1.RoleBinding {
	editRole := &rbacv1.ClusterRole{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: "edit"}, editRole)
	if err != nil {
		logger.Info("edit ClusterRole not found. Default rolebinding will not be created")
	}
	roleRef := rbacv1.RoleRef{
		Name:     editRole.GetName(),
		Kind:     editRole.Kind,
		APIGroup: "rbac.authorization.k8s.io",
	}
	roleBinding := resources.NewRoleBinding(jenkins, roleRef)
	return roleBinding
}
