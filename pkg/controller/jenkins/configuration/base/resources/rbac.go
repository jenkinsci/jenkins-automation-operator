package resources

import (
	"k8s.io/api/rbac/v1"
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
	//EmptyAPIGroup short hand for the empty API group while defining policies
	EmptyAPIGroup = ""
	//OpenshiftAPIGroup the openshift api group name
	OpenshiftAPIGroup = "image.openshift.io"
	//BuildAPIGroup  the openshift api group name for builds
	BuildAPIGroup = "build.openshift.io"
)

// NewRole returns rbac role for jenkins master
func NewRole(meta metav1.ObjectMeta) *v1.Role {
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
func NewRoleBinding(name, namespace, serviceAccountName string, roleRef v1.RoleRef) *v1.RoleBinding {
	return &v1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
	ReadOnly := []string{getVerb, listVerb, watchVerb}
	Default := []string{createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb}
	Create := []string{createVerb}

	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/portforward", Create))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/exec", Default))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "configmaps", ReadOnly))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "pods/log", ReadOnly))
	rules = append(rules, NewPolicyRule(EmptyAPIGroup, "secrets", ReadOnly))

	rules = append(rules, NewOpenShiftPolicyRule(OpenshiftAPIGroup, "imagestreams", ReadOnly))
	rules = append(rules, NewOpenShiftPolicyRule(BuildAPIGroup, "buildconfigs", ReadOnly))
	rules = append(rules, NewOpenShiftPolicyRule(BuildAPIGroup, "builds", ReadOnly))

	return rules
}

// NewPolicyRule returns a policyRule allowing verbs on resources
func NewPolicyRule(apiGroup string, resource string, verbs []string) v1.PolicyRule {
	rule := v1.PolicyRule{
		APIGroups: []string{apiGroup},
		Resources: []string{resource},
		Verbs:     verbs,
	}
	return rule
}

// NewOpenShiftPolicyRule returns a policyRule allowing verbs on resources
func NewOpenShiftPolicyRule(apiGroup string, resource string, verbs []string) v1.PolicyRule {
	return NewPolicyRule(apiGroup, resource, verbs)
}
