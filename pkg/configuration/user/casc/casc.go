package casc

import (
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"

	"github.com/go-logr/logr"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"k8s.io/client-go/rest"
)

const groovyUtf8MaxStringLength = 65535

// ConfigurationAsCode defines client for configurationAsCode
type ConfigurationAsCode interface {
	EnsureCasc(jenkinsName string) (requeue bool, err error)
	EnsureGroovy(jenkinsName string) (requeue bool, err error)
}

// TODO merge configurationAsCode struct and configuration.Configuration struct
// into a single one. See backuprestore.go for inheritance
type configurationAsCode struct {
	configuration.Configuration
	RestConfig   rest.Config
	Casc         *v1alpha3.Casc
	GroovyClient *groovy.Groovy
	Logger       logr.Logger
}

// New creates new instance of ConfigurationAsCode
func New(config configuration.Configuration, restConfig rest.Config, jenkinsClient jenkinsclient.Jenkins, configurationType string, casc *v1alpha3.Casc, customization v1alpha3.Customization) ConfigurationAsCode {
	return &configurationAsCode{
		Configuration: config,
		Casc:          casc,
		RestConfig:    restConfig,
		GroovyClient:  groovy.New(jenkinsClient, config.Client, casc, configurationType, customization),
		Logger:        log.Log.WithValues("cr", casc.Name),
	}
}

// EnsureCasc configures Jenkins with help Configuration as a code plugin
func (c *configurationAsCode) EnsureCasc(jenkinsName string) (requeue bool, err error) {
	//Add Labels to secret
	namespace := c.Casc.ObjectMeta.Namespace
	secretName := c.Casc.Spec.ConfigurationAsCode.Secret.Name
	if err := resources.AddLabelToWatchedSecrets(jenkinsName, secretName, namespace, c.Client); err != nil {
		c.Logger.V(log.VDebug).Info(fmt.Sprintf("Error while adding labels to secret '%s' : %s", secretName, err))
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configurationAsCode secret")

	//Add Labels to configmaps
	configMaps := c.Casc.Spec.ConfigurationAsCode.Configurations
	if err := resources.AddLabelToWatchedCMs(jenkinsName, namespace, c.Client, configMaps); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info(fmt.Sprintf("labels added to configurationAsCode configmap: '%+v'", configMaps))
	podName := c.GetJenkinsMasterPodName()
	// Reconcile
	c.Logger.V(log.VDebug).Info(fmt.Sprintf("Copying secret '%s' from pod to pod's filesystem using restConfig: %+v ", secretName, c))
	requeue, err = resources.CopySecret(c.Client, c.ClientSet, c.RestConfig, podName, secretName, namespace)
	if err != nil || requeue {
		return requeue, err
	}

	hasYamlSuffix := func(name string) bool {
		return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
	}
	prepareScript := func(groovyScript string) string {
		return fmt.Sprintf(applyConfigurationAsCodeGroovyScriptFmt, prepareScript(groovyScript))
	}
	return c.GroovyClient.Ensure(hasYamlSuffix, prepareScript)
}

// EnsureCasc configures Jenkins with help Configuration as a code plugin
func (c *configurationAsCode) EnsureGroovy(jenkinsCRName string) (requeue bool, err error) {
	//Add Labels to secret
	if err := resources.AddLabelToWatchedSecrets(jenkinsCRName, c.Casc.Spec.GroovyScripts.Secret.Name, c.Casc.ObjectMeta.Namespace, c.Client); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configuration as code secret")

	//Add Labels to configmaps
	if err := resources.AddLabelToWatchedCMs(jenkinsCRName, c.Casc.ObjectMeta.Namespace, c.Client, c.Casc.Spec.GroovyScripts.Configurations); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configuration as code configmap")

	podName := c.GetJenkinsMasterPodName()
	// Reconcile
	requeue, err = resources.CopySecret(c.Client, c.ClientSet, c.RestConfig, podName, c.Casc.Spec.ConfigurationAsCode.Secret.Name, c.Casc.ObjectMeta.Namespace)
	if err != nil || requeue {
		return requeue, err
	}

	return c.GroovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".groovy")
	}, groovy.AddSecretsLoaderToGroovyScript())
}

const applyConfigurationAsCodeGroovyScriptFmt = `
String[] configContent = ['''%s''']

def configSb = new StringBuffer()
for (int i=0; i<configContent.size(); i++) {
    configSb << configContent[i]
}

def stream = new ByteArrayInputStream(configSb.toString().getBytes('UTF-8'))
def source = io.jenkins.plugins.casc.yaml.YamlSource.of(stream)

io.jenkins.plugins.casc.ConfigurationAsCode.get().configureWith(source)
`

func prepareScript(script string) string {
	var slicedScript []string
	if len(script) > groovyUtf8MaxStringLength {
		slicedScript = splitTooLongScript(script)
	} else {
		slicedScript = append(slicedScript, script)
	}

	return strings.Join(slicedScript, "''','''")
}

func splitTooLongScript(groovyScript string) []string {
	var slicedGroovyScript []string

	lastSubstrIndex := len(groovyScript) % groovyUtf8MaxStringLength
	lastSubstr := groovyScript[len(groovyScript)-lastSubstrIndex:]

	substrNumber := len(groovyScript) / groovyUtf8MaxStringLength
	for i := 0; i < substrNumber; i++ {
		scriptSubstr := groovyScript[i*groovyUtf8MaxStringLength : (i*groovyUtf8MaxStringLength)+groovyUtf8MaxStringLength]
		slicedGroovyScript = append(slicedGroovyScript, scriptSubstr)
	}

	slicedGroovyScript = append(slicedGroovyScript, lastSubstr)

	return slicedGroovyScript
}
