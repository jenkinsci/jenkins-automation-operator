package casc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	userConfigurationHashParameterName       = "userConfigurationHash"
	userConfigurationSecretHashParameterName = "userConfigurationSecretHash"
)

// ConfigurationAsCode defines API which configures Jenkins with help Configuration as a code plugin
type ConfigurationAsCode struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
	jobName       string
}

// New creates new instance of ConfigurationAsCode
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger, jobName string) *ConfigurationAsCode {
	return &ConfigurationAsCode{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
		jobName:       jobName,
	}
}

// ConfigureJob configures jenkins job which configures Jenkins with help Configuration as a code plugin
func (g *ConfigurationAsCode) ConfigureJob() error {
	_, created, err := g.jenkinsClient.CreateOrUpdateJob(
		fmt.Sprintf(configurationJobXMLFmt, resources.UserConfigurationSecretVolumePath, resources.JenkinsUserConfigurationVolumePath),
		g.jobName)
	if err != nil {
		return err
	}
	if created {
		g.logger.Info(fmt.Sprintf("'%s' job has been created", g.jobName))
	}
	return nil
}

// Ensure configures Jenkins with help Configuration as a code plugin
func (g *ConfigurationAsCode) Ensure(jenkins *v1alpha1.Jenkins) (bool, error) {
	jobsClient := jobs.New(g.jenkinsClient, g.k8sClient, g.logger)

	configuration := &corev1.ConfigMap{}
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetUserConfigurationConfigMapNameFromJenkins(jenkins)}
	err := g.k8sClient.Get(context.TODO(), namespaceName, configuration)
	if err != nil {
		return false, errors.WithStack(err)
	}

	secret := &corev1.Secret{}
	namespaceName = types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetUserConfigurationSecretNameFromJenkins(jenkins)}
	err = g.k8sClient.Get(context.TODO(), namespaceName, configuration)
	if err != nil {
		return false, errors.WithStack(err)
	}

	userConfigurationSecretHash := g.calculateUserConfigurationSecretHash(secret)
	userConfigurationHash := g.calculateUserConfigurationHash(configuration)
	done, err := jobsClient.EnsureBuildJob(
		g.jobName,
		userConfigurationSecretHash+userConfigurationHash,
		map[string]string{
			userConfigurationHashParameterName:       userConfigurationHash,
			userConfigurationSecretHashParameterName: userConfigurationSecretHash,
		},
		jenkins,
		true)
	if err != nil {
		return false, err
	}
	return done, nil
}

func (g *ConfigurationAsCode) calculateUserConfigurationSecretHash(userConfigurationSecret *corev1.Secret) string {
	hash := sha256.New()

	var keys []string
	for key := range userConfigurationSecret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte(userConfigurationSecret.Data[key]))
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (g *ConfigurationAsCode) calculateUserConfigurationHash(userConfiguration *corev1.ConfigMap) string {
	hash := sha256.New()

	var keys []string
	for key := range userConfiguration.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasSuffix(key, ".yaml") {
			hash.Write([]byte(key))
			hash.Write([]byte(userConfiguration.Data[key]))
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
          <name>` + userConfigurationSecretHashParameterName + `</name>
          <description/>
          <defaultValue/>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + userConfigurationHashParameterName + `</name>
          <description/>
          <defaultValue/>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61.1">
    <script>import io.jenkins.plugins.casc.yaml.YamlSource;

def secretsPath = &apos;%s&apos;
def configsPath = &apos;%s&apos;
def userConfigurationSecretExpectedHash = params.` + userConfigurationSecretHashParameterName + `
def userConfigurationExpectedHash = params.` + userConfigurationHashParameterName + `

node(&apos;master&apos;) {
    def secretsText = sh(script: &quot;ls ${secretsPath} | grep .yaml | sort&quot;, returnStdout: true).trim()
    def secrets = []
    secrets.addAll(secretsText.tokenize(&apos;\n&apos;))

    def configsText = sh(script: &quot;ls ${configsPath} | grep .yaml | sort&quot;, returnStdout: true).trim()
    def configs = []
    configs.addAll(configsText.tokenize(&apos;\n&apos;))
    
    stage(&apos;Synchronizing files&apos;) {
		synchronizeFiles(secretsPath, (String[])secrets, userConfigurationSecretExpectedHash)
		synchronizeFiles(configsPath, (String[])configs, userConfigurationExpectedHash)
    }
    
    for(config in configs) {
        stage(config) {
            def path = java.nio.file.Paths.get(&quot;${configsPath}/${config}&quot;)
            def source = new YamlSource(path, YamlSource.READ_FROM_PATH)
            io.jenkins.plugins.casc.ConfigurationAsCode.get().configureWith(source)
        }
    }
}

def synchronizeFiles(String path, String[] files, String hash) {
    def complete = false
    for(int i = 1; i &lt;= 10; i++) {
        def actualHash = calculateHash(files, path)
        println &quot;Expected hash &apos;${hash}&apos;, actual hash &apos;${actualHash}&apos;, path &apos;${path}&apos;&quot;
        if(hash == actualHash) {
            complete = true
            break
        }
        sleep 2
    }
    if(!complete) {
        error(&quot;Timeout while synchronizing files&quot;)
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
