package casc

import (
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigurationAsCode defines API which configures Jenkins with help Configuration as a code plugin
type ConfigurationAsCode struct {
	groovyClient *groovy.Groovy
}

// New creates new instance of ConfigurationAsCode
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger, jenkins *v1alpha2.Jenkins) *ConfigurationAsCode {
	return &ConfigurationAsCode{
		groovyClient: groovy.New(jenkinsClient, k8sClient, logger, jenkins, "user-casc", jenkins.Spec.ConfigurationAsCode.Customization),
	}
}

// Ensure configures Jenkins with help Configuration as a code plugin
func (c *ConfigurationAsCode) Ensure(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	requeue, err = c.groovyClient.WaitForSecretSynchronization(resources.ConfigurationAsCodeSecretVolumePath)
	if err != nil || requeue {
		return requeue, err
	}

	return c.groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
	}, func(groovyScript string) string {
		return fmt.Sprintf(applyConfigurationAsCodeGroovyScriptFmt, prepareScript(groovyScript))
	})
}

const applyConfigurationAsCodeGroovyScriptFmt = `
String[] configContent = ['''%s'''] 
def configSb = new StringBuffer()
for (int i=0; i<configContent.size(); i++) {
    configSb << configContent[i]
}

def stream = new ByteArrayInputStream(configSb.toString().getBytes('UTF-8'))

def source = new io.jenkins.plugins.casc.yaml.YamlSource(stream, io.jenkins.plugins.casc.yaml.YamlSource.READ_FROM_INPUTSTREAM)
io.jenkins.plugins.casc.ConfigurationAsCode.get().configureWith(source)
`

func prepareScript(script string) string {
	groovyUtf8MaxStringLength := 65535
	var slicedScript []string
	if len(script) > groovyUtf8MaxStringLength {
		slicedScript = splitTooLongScript(script)
	} else {
		slicedScript = append(slicedScript, script)
	}

	return strings.Join(slicedScript, "''','''")
}

func splitTooLongScript(groovyScript string) []string {
	groovyUtf8MaxStringLength := 65535
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
