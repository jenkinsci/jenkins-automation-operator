package groovy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	//stackerr "github.com/pkg/errors"
)

//nolint: golint
// GroovyIface
type GroovyIface interface {
	GetNamespace() string
	GetCRName() string
}

// Groovy defines API for groovy secrets execution via jenkins job
type Groovy struct {
	k8sClient         k8s.Client
	jenkinsClient     jenkinsclient.Jenkins
	customization     v1alpha3.Customization
	groovyIface       GroovyIface
	configurationType string
	logger            logr.Logger
}

// New creates new instance of Groovy
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, groovyIface GroovyIface, configurationType string,
	customization v1alpha3.Customization) *Groovy {
	return &Groovy{
		jenkinsClient:     jenkinsClient,
		k8sClient:         k8sClient,
		customization:     customization,
		groovyIface:       groovyIface,
		configurationType: configurationType,
		logger:            log.Log.WithValues("cr", "jenkins"),
	}
}

// EnsureSingle runs single groovy script
func (g *Groovy) EnsureSingle(source, scriptName, hash, groovyScript string) (requeue bool, err error) {
	scriptType := g.configurationType
	if g.isGroovyScriptAlreadyApplied(source, scriptName, hash, scriptType) {
		g.logger.V(log.VWarn).Info(fmt.Sprintf("Groovy script '%s' has been already applied: skipping", scriptName))
		return false, nil
	}
	g.logger.V(log.VWarn).Info(fmt.Sprintf("About to execute groovy scripts using jenkinsClient: %s", g.jenkinsClient))
	g.logger.V(log.VWarn).Info(fmt.Sprintf("Triggering execution of groovy script '%s'", scriptName))
	logs, err := g.jenkinsClient.ExecuteScript(groovyScript)

	g.logger.V(log.VWarn).Info(fmt.Sprintf("Logs for groovy script execution '%s'", logs))
	if err != nil {
		if groovyErr, ok := err.(*jenkinsclient.GroovyScriptExecutionFailed); ok {
			groovyErr.ConfigurationType = scriptType
			groovyErr.Name = scriptName
			groovyErr.Source = source
			groovyErr.Logs = logs
			g.logger.V(log.VWarn).Info(fmt.Sprintf("Execution of script type: '%s', named: '%s' from configmap: '%s' failed with the following logs:\n%s", scriptType, scriptName, source, logs))
		}
		g.logger.V(log.VWarn).Info(fmt.Sprintf("Successful execution of script type: '%s', named: '%s' from configmap: '%s'", scriptType, scriptName, source))
		return true, err
	}
	// Add the script to appliedGroovyScripts in the right object
	switch v := g.groovyIface.(type) {
	case *v1alpha3.Casc:
		var appliedGroovyScripts []v1alpha3.AppliedGroovyScript
		for _, ags := range v.Status.AppliedGroovyScripts {
			if ags.ConfigurationType == scriptType && ags.Source == source && ags.Name == scriptName {
				continue
			}
			appliedGroovyScripts = append(appliedGroovyScripts, ags)
		}
		appliedGroovyScripts = append(appliedGroovyScripts, v1alpha3.AppliedGroovyScript{
			ConfigurationType: scriptType,
			Source:            source,
			Name:              scriptName,
			Hash:              hash,
		})
		v.Status.AppliedGroovyScripts = appliedGroovyScripts
		err = g.k8sClient.Update(context.TODO(), v)
		if err != nil {
			return true, err
		}
	case *v1alpha2.Jenkins:
		var appliedGroovyScripts []v1alpha2.AppliedGroovyScript
		for _, ags := range v.Status.AppliedGroovyScripts {
			if ags.ConfigurationType == scriptType && ags.Source == source && ags.Name == scriptName {
				continue
			}
			appliedGroovyScripts = append(appliedGroovyScripts, ags)
		}
		appliedGroovyScripts = append(appliedGroovyScripts, v1alpha2.AppliedGroovyScript{
			ConfigurationType: scriptType,
			Source:            source,
			Name:              scriptName,
			Hash:              hash,
		})
		v.Status.AppliedGroovyScripts = appliedGroovyScripts
		err = g.k8sClient.Update(context.TODO(), v)
		if err != nil {
			return true, err
		}
	}
	return true, nil
}

/* WaitForSecretSynchronization runs groovy script which waits to synchronize secrets in pod by k8s
func (g *Groovy) WaitForSecretSynchronization(secretsPath string) (requeue bool, err error) {
	if len(g.customization.Secret.Name) == 0 {
		return false, nil
	}

	secret := &corev1.Secret{}
	err = g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: g.customization.Secret.Name, g.groovyIface.GetNamespace()}, secret)
	if err != nil {
		return true, errors.WithStack(err)
	}

	toCalculate := map[string]string{}
	for secretKey, secretValue := range secret.Data {
		toCalculate[secretKey] = string(secretValue)
	}
	hash := g.calculateHash(toCalculate)

	name := "synchronizing-secret.groovy"

	g.logger.Info(fmt.Sprintf("%s Secret '%s' running synchronization", g.configurationType, secret.Name))
	return g.EnsureSingle(g.customization.Secret.Name, name, hash, fmt.Sprintf(synchronizeSecretsGroovyScriptFmt, secretsPath, hash))
}
*/

// Ensure runs all groovy scripts configured in customization structure
func (g *Groovy) Ensure(filter func(name string) bool, updateGroovyScript func(groovyScript string) string) (requeue bool, err error) {
	secret := &corev1.Secret{}
	if len(g.customization.Secret.Name) > 0 {
		err := g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: g.customization.Secret.Name, Namespace: g.groovyIface.GetNamespace()}, secret)
		if err != nil {
			return true, err
		}
	}

	for _, configMapRef := range g.customization.Configurations {
		configMap := &corev1.ConfigMap{}
		err := g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: g.groovyIface.GetNamespace()}, configMap)
		if err != nil {
			return true, errors.WithStack(err)
		}

		var names []string
		for name := range configMap.Data {
			names = append(names, name)
		}
		sort.Strings(names)
		// FIXME check that scripts are listed in right order ?
		for _, name := range names {
			groovyScript := updateGroovyScript(configMap.Data[name])
			if !filter(name) {
				g.logger.V(log.VDebug).Info(fmt.Sprintf("Skipping %s ConfigMap '%s' name '%s'", g.configurationType, configMap.Name, name))
				continue
			}
			hash := g.calculateCustomizationHash(*secret, name, groovyScript)

			g.logger.Info(fmt.Sprintf("%s ConfigMap '%s' name '%s' running groovy script", g.configurationType, configMap.Name, name))
			requeue, err := g.EnsureSingle(configMap.Name, name, hash, groovyScript)
			if err != nil || requeue {
				return requeue, err
			}
		}
	}

	return false, nil
}

func (g *Groovy) calculateCustomizationHash(secret corev1.Secret, key, groovyScript string) string {
	toCalculate := map[string]string{}
	for secretKey, secretValue := range secret.Data {
		toCalculate[secretKey] = string(secretValue)
	}
	toCalculate[key] = groovyScript
	return g.calculateHash(toCalculate)
}

func (g *Groovy) isGroovyScriptAlreadyApplied(source, name, hash, configurationType string) bool {
	type S struct{}
	switch v := g.groovyIface.(type) {
	case *v1alpha3.Casc:
		script := v1alpha3.AppliedGroovyScript{
			ConfigurationType: configurationType,
			Source:            source,
			Name:              name,
			Hash:              hash,
		}
		appliedGroovyScripts := v.Status.AppliedGroovyScripts
		applied := make(map[v1alpha3.AppliedGroovyScript]struct{}, len(appliedGroovyScripts))
		for _, script := range appliedGroovyScripts {
			applied[script] = S{}
		}
		if _, ok := applied[script]; ok {
			return true
		}
	case *v1alpha2.Jenkins:
		script := v1alpha2.AppliedGroovyScript{
			ConfigurationType: configurationType,
			Source:            source,
			Name:              name,
			Hash:              hash,
		}
		appliedGroovyScripts := v.Status.AppliedGroovyScripts
		applied := make(map[v1alpha2.AppliedGroovyScript]struct{}, len(appliedGroovyScripts))
		for _, script := range appliedGroovyScripts {
			applied[script] = S{}
		}
		if _, ok := applied[script]; ok {
			return true
		}
	}

	return false
}

func (g *Groovy) calculateHash(data map[string]string) string {
	hash := sha256.New()

	var keys []string
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte(data[key]))
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// AddSecretsLoaderToGroovyScript modify groovy scripts to load Kubernetes secrets into groovy map
func AddSecretsLoaderToGroovyScript() func(groovyScript string) string {
	return func(groovyScript string) string {
		return groovyScript + loaderGroovyScriptFmt
	}
}

const loaderGroovyScriptFmt = `
import io.jenkins.plugins.casc.ConfigurationAsCode;
ConfigurationAsCode.get().configure()
`
