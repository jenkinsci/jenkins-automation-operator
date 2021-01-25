package resources

import (
	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createVerb = "create"
	deleteVerb = "delete"
	getVerb    = "get"
	listVerb   = "list"
	watchVerb  = "watch"
	patchVerb  = "patch"
	updateVerb = "update"
	useVerb    = "use"

	// EmptyAPIGroup short hand for the empty API group while defining policies
	EmptyAPIGroup = ""

	// ImageAPIGroup the openshift api group name for images
	ImageAPIGroup = "image.openshift.io"

	// BuildAPIGroup  the openshift api group name for builds
	BuildAPIGroup = "build.openshift.io"

	// SecurityAPIGroup  the openshift api group name for security
	SecurityAPIGroup = "security.openshift.io"
)

// NewRole returns rbac role for jenkins master
func NewRole(jenkins *v1alpha2.Jenkins) *v1.Role {
	meta := NewResourceObjectMeta(jenkins)
	rules := NewDefaultPolicyRules()
	return &v1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta,
		Rules:      rules,
	}
}

// NewRoleBinding returns rbac role binding for jenkins master
func NewRoleBinding(jenkins *v1alpha2.Jenkins, roleRef v1.RoleRef) *v1.RoleBinding {
	resourceWithPrefix := NewResourceObjectMeta(jenkins)
	roleBindingName := GetExtraRoleBindingName(jenkins, roleRef)
	serviceAccountName := resourceWithPrefix.Name
	namespace := jenkins.Namespace
	return &v1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: namespace,
		},
		RoleRef: roleRef,
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
	}
}

// NewDefaultPolicyRules sets the default policy rules
func NewDefaultPolicyRules() []v1.PolicyRule {
	var rules []v1.PolicyRule
	Default := []string{createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb}
	create := []string{createVerb}
	watch := []string{watchVerb}
	use := []string{useVerb}

	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/portforward", "", create))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods", "", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/exec", "", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "configmaps", "", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/log", "", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "secrets", "", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "events", "", watch))

	rules = append(rules, NewPolicyRule(ImageAPIGroup, "imagestreams", "", Default))
	rules = append(rules, NewPolicyRule(BuildAPIGroup, "buildconfigs", "", Default))
	rules = append(rules, NewPolicyRule(BuildAPIGroup, "builds", "", Default))
	rules = append(rules, NewPolicyRule(SecurityAPIGroup, "securitycontextconstraints", "anyuid", use))

	return rules
}

// NewPolicyRule returns a policyRule allowing verbs on resources
func NewPolicyRule(apiGroup string, resource, resourceName string, verbs []string) v1.PolicyRule {
	rule := v1.PolicyRule{
		APIGroups:     []string{apiGroup},
		Resources:     []string{resource},
		ResourceNames: []string{resourceName},
		Verbs:         verbs,
	}
	return rule
}
