package resources

import (
	"context"

	apiconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	openshiftGlobalProxy = "cluster"
	envHTTPProxyName     = "http_proxy"
	envHTTPSProxyName    = "https_proxy"
	envNoProxyName       = "no_proxy"
)

var (
	ProxyAPIAvailable   = false
	ProxyAPIChecked     = false
	allProxyEnvVarNames = []string{
		envHTTPProxyName,
		envHTTPSProxyName,
		envNoProxyName,
	}
	ProxyEnvVars = []corev1.EnvVar{}
	GlobalProxy  = &apiconfigv1.Proxy{}
)

func IsOpenshiftProxyAPIAvailable(clientSet *kubernetes.Clientset) bool {
	if ProxyAPIChecked {
		return ProxyAPIAvailable
	}
	gv := schema.GroupVersion{
		Group:   apiconfigv1.GroupName,
		Version: apiconfigv1.SchemeGroupVersion.Version,
	}
	if err := discovery.ServerSupportsVersion(clientSet, gv); err != nil {
		// error, API not available
		ProxyAPIChecked = true
		ProxyAPIAvailable = false
	} else {
		// API Exists
		ProxyAPIChecked = true
		ProxyAPIAvailable = true
	}
	return ProxyAPIAvailable
}

func QueryProxyConfig(manager manager.Manager) error {
	err := manager.GetAPIReader().Get(context.TODO(), client.ObjectKey{Name: openshiftGlobalProxy}, GlobalProxy)
	if err != nil {
		logger.Info("openshift global proxy not found")
		return err
	}

	// We have found the global proxy configuration object!
	ProxyEnvVars = ToEnvVar(GlobalProxy)

	return nil
}

func ToEnvVar(proxy *apiconfigv1.Proxy) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  envHTTPProxyName,
			Value: proxy.Status.HTTPProxy,
		},
		{
			Name:  envHTTPSProxyName,
			Value: proxy.Status.HTTPSProxy,
		},
		{
			Name:  envNoProxyName,
			Value: proxy.Status.NoProxy,
		},
	}
}

// IsOverridden returns true if the given container overrides proxy env variable(s).
// We apply the following rule:
//   If a container already defines any of the proxy env variable then it
//   overrides all of these.
func IsProxyOverridden(envVar []corev1.EnvVar) (overrides bool) {
	for _, envVarName := range allProxyEnvVarNames {
		_, found := findProxyEnvVar(envVar, envVarName)
		if found {
			overrides = true
			return
		}
	}

	return
}

func findProxyEnvVar(proxyEnvVar []corev1.EnvVar, name string) (envVar *corev1.EnvVar, found bool) {
	for i := range proxyEnvVar {
		if name == proxyEnvVar[i].Name {
			// Environment variable names are case sensitive.
			found = true
			envVar = &proxyEnvVar[i]

			break
		}
	}

	return
}

func dropEmptyProxyEnv(in []corev1.EnvVar) (out []corev1.EnvVar) {
	out = make([]corev1.EnvVar, 0)
	for i := range in {
		if in[i].Value == "" {
			continue
		}

		out = append(out, in[i])
	}

	return
}
