package casc

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha3"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const groovyUtf8MaxStringLength = 65535

// ConfigurationAsCode defines client for configurationAsCode
type ConfigurationAsCode interface {
	EnsureCasc(jenkinsName string) (requeue bool, err error)
	EnsureGroovy(jenkinsName string) (requeue bool, err error)
}

type configurationAsCode struct {
	Casc         *v1alpha3.Casc
	GroovyClient *groovy.Groovy
	K8sClient    k8s.Client
	ClientSet    kubernetes.Clientset
	RestConfig   *rest.Config
	Logger       logr.Logger
}

// New creates new instance of ConfigurationAsCode
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, clientSet kubernetes.Clientset, restConfig *rest.Config, configurationType string, casc *v1alpha3.Casc, customization v1alpha3.Customization) ConfigurationAsCode {
	return &configurationAsCode{
		GroovyClient: groovy.New(jenkinsClient, k8sClient, casc, configurationType, customization),
		Casc:         casc,
		K8sClient:    k8sClient,
		ClientSet:    clientSet,
		RestConfig:   restConfig,
		Logger:       log.Log.WithValues("cr", casc.Name),
	}
}

// EnsureCasc configures Jenkins with help Configuration as a code plugin
func (c *configurationAsCode) EnsureCasc(jenkinsName string) (requeue bool, err error) {
	//Add Labels to secret
	if err := resources.AddLabelToWatchedSecrets(jenkinsName, c.Casc.Spec.ConfigurationAsCode.Secret.Name, c.Casc.ObjectMeta.Namespace, c.K8sClient); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configurationAsCode secret")

	//Add Labels to configmaps
	if err := resources.AddLabelToWatchedCMs(jenkinsName, c.Casc.ObjectMeta.Namespace, c.K8sClient, c.Casc.Spec.ConfigurationAsCode.Configurations); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configurationAsCode configmap")
	// Reconcile
	requeue, err = resources.CopySecret(c.K8sClient, c.ClientSet, c.RestConfig, resources.GetJenkinsMasterPodName(jenkinsName), c.Casc.Spec.ConfigurationAsCode.Secret.Name, c.Casc.ObjectMeta.Namespace)
	if err != nil || requeue {
		return requeue, err
	}

	return c.GroovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
	}, func(groovyScript string) string {
		return fmt.Sprintf(applyConfigurationAsCodeGroovyScriptFmt, prepareScript(groovyScript))
	})
}

// EnsureCasc configures Jenkins with help Configuration as a code plugin
func (c *configurationAsCode) EnsureGroovy(jenkinsName string) (requeue bool, err error) {
	//Add Labels to secret
	if err := resources.AddLabelToWatchedSecrets(jenkinsName, c.Casc.Spec.GroovyScripts.Secret.Name, c.Casc.ObjectMeta.Namespace, c.K8sClient); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configuration as conde secret")

	//Add Labels to configmaps
	if err := resources.AddLabelToWatchedCMs(jenkinsName, c.Casc.ObjectMeta.Namespace, c.K8sClient, c.Casc.Spec.GroovyScripts.Configurations); err != nil {
		return true, err
	}
	c.Logger.V(log.VDebug).Info("labels added to configuration as code configmap")

	// Reconcile
	requeue, err = resources.CopySecret(c.K8sClient, c.ClientSet, c.RestConfig, resources.GetJenkinsMasterPodName(jenkinsName), c.Casc.Spec.ConfigurationAsCode.Secret.Name, c.Casc.ObjectMeta.Namespace)
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
