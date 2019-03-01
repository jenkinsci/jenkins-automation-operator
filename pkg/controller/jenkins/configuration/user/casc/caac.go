package casc

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	jobHashParameterName = "hash"
)

// ConfigurationAsCode defines API which configures Jenkins with help Configuration as a code plugin
type ConfigurationAsCode struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
	jobName       string
	configsPath   string
}

// New creates new instance of ConfigurationAsCode
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger, jobName, configsPath string) *ConfigurationAsCode {
	return &ConfigurationAsCode{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
		jobName:       jobName,
		configsPath:   configsPath,
	}
}

// ConfigureJob configures jenkins job which configures Jenkins with help Configuration as a code plugin
func (g *ConfigurationAsCode) ConfigureJob() error {
	_, created, err := g.jenkinsClient.CreateOrUpdateJob(fmt.Sprintf(configurationJobXMLFmt, g.configsPath), g.jobName)
	if err != nil {
		return err
	}
	if created {
		g.logger.Info(fmt.Sprintf("'%s' job has been created", g.jobName))
	}
	return nil
}

// Ensure configures Jenkins with help Configuration as a code plugin
func (g *ConfigurationAsCode) Ensure(secretOrConfigMapData map[string]string, jenkins *v1alpha1.Jenkins) (bool, error) {
	jobsClient := jobs.New(g.jenkinsClient, g.k8sClient, g.logger)

	hash := g.calculateHash(secretOrConfigMapData)
	done, err := jobsClient.EnsureBuildJob(g.jobName, hash, map[string]string{jobHashParameterName: hash}, jenkins, true)
	if err != nil {
		return false, err
	}
	return done, nil
}

func (g *ConfigurationAsCode) calculateHash(secretOrConfigMapData map[string]string) string {
	hash := sha256.New()

	var keys []string
	for key := range secretOrConfigMapData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasSuffix(key, ".yaml") {
			hash.Write([]byte(key))
			hash.Write([]byte(secretOrConfigMapData[key]))
		}
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

const configurationJobXMLFmt = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.31">
  <actions/>
  <description></description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty/>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.StringParameterDefinition>
          <name>` + jobHashParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61.1">
    <script>import io.jenkins.plugins.casc.yaml.YamlSource;

def configsPath = &apos;%s&apos;
def expectedHash = params.hash

node(&apos;master&apos;) {
    def configsText = sh(script: &quot;ls ${configsPath} | grep .yaml | sort&quot;, returnStdout: true).trim()
    def configs = []
    configs.addAll(configsText.tokenize(&apos;\n&apos;))
    
    stage(&apos;Synchronizing files&apos;) {
        def complete = false
        for(int i = 1; i &lt;= 10; i++) {
            def actualHash = calculateHash((String[])configs, configsPath)
            println &quot;Expected hash &apos;${expectedHash}&apos;, actual hash &apos;${actualHash}&apos;&quot;
            if(expectedHash == actualHash) {
                complete = true
                break
            }
            sleep 2
        }
        if(!complete) {
            error(&quot;Timeout while synchronizing files&quot;)
        }
    }
    
    for(config in configs) {
        stage(config) {
            def path = java.nio.file.Paths.get(&quot;${configsPath}/${config}&quot;)
            def source = new YamlSource(path, YamlSource.READ_FROM_PATH)
            io.jenkins.plugins.casc.ConfigurationAsCode.get().configureWith(source)
        }
    }
}

@NonCPS
def calculateHash(String[] configs, String configsPath) {
    def hash = java.security.MessageDigest.getInstance(&quot;SHA-256&quot;)
    for(config in configs) {
        hash.update(config.getBytes())
        def fileLocation = java.nio.file.Paths.get(&quot;${configsPath}/${config}&quot;)
        def fileData = java.nio.file.Files.readAllBytes(fileLocation)
        hash.update(fileData)
    }
    return Base64.getEncoder().encodeToString(hash.digest())
}</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
