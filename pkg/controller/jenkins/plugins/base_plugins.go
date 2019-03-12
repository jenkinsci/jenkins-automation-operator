package plugins

const (
	apacheComponentsClientPlugin = "apache-httpcomponents-client-4-api:4.5.5-3.0"
	jackson2ADIPlugin            = "jackson2-api:2.9.8"
	credentialsPlugin            = "credentials:2.1.18"
	cloudBeesFolderPlugin        = "cloudbees-folder:6.7"
	durableTaskPlugin            = "durable-task:1.29"
	plainCredentialsPlugin       = "plain-credentials:1.5"
	structsPlugin                = "structs:1.17"
	workflowStepAPIPlugin        = "workflow-step-api:2.19"
	scmAPIPlugin                 = "scm-api:2.3.0"
	workflowAPIPlugin            = "workflow-api:2.33"
	workflowSupportPlugin        = "workflow-support:3.2"
	displayURLAPIPlugin          = "display-url-api:2.3.0"
	gitClientPlugin              = "git-client:2.7.6"
	jschPlugin                   = "jsch:0.1.55"
	junitPlugin                  = "junit:1.27"
	mailerPlugin                 = "mailer:1.23"
	matrixProjectPlugin          = "matrix-project:1.14"
	scriptSecurityPlugin         = "script-security:1.54"
	sshCredentialsPlugin         = "ssh-credentials:1.15"
	workflowSCMStepPlugin        = "workflow-scm-step:2.7"
	variantPlugin                = "variant:1.2"
)

// BasePluginsMap contains plugins to install by operator
var BasePluginsMap = map[string][]Plugin{
	Must(New("kubernetes:1.14.8")).String(): {
		Must(New(apacheComponentsClientPlugin)),
		Must(New(cloudBeesFolderPlugin)),
		Must(New(credentialsPlugin)),
		Must(New(durableTaskPlugin)),
		Must(New(jackson2ADIPlugin)),
		Must(New("kubernetes-credentials:0.4.0")),
		Must(New(plainCredentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(variantPlugin)),
		Must(New(workflowStepAPIPlugin)),
	},
	Must(New("workflow-job:2.32")).String(): {
		Must(New(scmAPIPlugin)),
		Must(New(scriptSecurityPlugin)),
		Must(New(structsPlugin)),
		Must(New(workflowAPIPlugin)),
		Must(New(workflowStepAPIPlugin)),
		Must(New(workflowSupportPlugin)),
	},
	Must(New("workflow-aggregator:2.6")).String(): {
		Must(New("ace-editor:1.1")),
		Must(New(apacheComponentsClientPlugin)),
		Must(New("authentication-tokens:1.3")),
		Must(New("branch-api:2.1.2")),
		Must(New(cloudBeesFolderPlugin)),
		Must(New("credentials-binding:1.18")),
		Must(New(credentialsPlugin)),
		Must(New(displayURLAPIPlugin)),
		Must(New("docker-commons:1.13")),
		Must(New("docker-workflow:1.17")),
		Must(New(durableTaskPlugin)),
		Must(New(gitClientPlugin)),
		Must(New("git-server:1.7")),
		Must(New("handlebars:1.1.1")),
		Must(New(jackson2ADIPlugin)),
		Must(New("jquery-detached:1.2.1")),
		Must(New(jschPlugin)),
		Must(New(junitPlugin)),
		Must(New("lockable-resources:2.4")),
		Must(New(mailerPlugin)),
		Must(New(matrixProjectPlugin)),
		Must(New("momentjs:1.1.1")),
		Must(New("pipeline-build-step:2.7")),
		Must(New("pipeline-graph-analysis:1.9")),
		Must(New("pipeline-input-step:2.9")),
		Must(New("pipeline-milestone-step:1.3.1")),
		Must(New("pipeline-model-api:1.3.6")),
		Must(New("pipeline-model-declarative-agent:1.1.1")),
		Must(New("pipeline-model-definition:1.3.6")),
		Must(New("pipeline-model-extensions:1.3.6")),
		Must(New("pipeline-rest-api:2.10")),
		Must(New("pipeline-stage-step:2.3")),
		Must(New("pipeline-stage-tags-metadata:1.3.6")),
		Must(New("pipeline-stage-view:2.10")),
		Must(New(plainCredentialsPlugin)),
		Must(New(scmAPIPlugin)),
		Must(New(scriptSecurityPlugin)),
		Must(New(sshCredentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(workflowAPIPlugin)),
		Must(New("workflow-basic-steps:2.14")),
		Must(New("workflow-cps-global-lib:2.13")),
		Must(New("workflow-cps:2.64")),
		Must(New("workflow-durable-task-step:2.29")),
		Must(New("workflow-job:2.32")),
		Must(New("workflow-multibranch:2.21")),
		Must(New(workflowSCMStepPlugin)),
		Must(New(workflowStepAPIPlugin)),
		Must(New(workflowSupportPlugin)),
	},
	Must(New("git:3.9.3")).String(): {
		Must(New(apacheComponentsClientPlugin)),
		Must(New(credentialsPlugin)),
		Must(New(displayURLAPIPlugin)),
		Must(New(gitClientPlugin)),
		Must(New(jschPlugin)),
		Must(New(junitPlugin)),
		Must(New(mailerPlugin)),
		Must(New(matrixProjectPlugin)),
		Must(New(scmAPIPlugin)),
		Must(New(scriptSecurityPlugin)),
		Must(New(sshCredentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(workflowAPIPlugin)),
		Must(New(workflowSCMStepPlugin)),
		Must(New(workflowStepAPIPlugin)),
	},
	Must(New("job-dsl:1.72")).String(): {
		Must(New(scriptSecurityPlugin)),
		Must(New(structsPlugin)),
	},
	Must(New("configuration-as-code:1.7")).String(): {
		Must(New("configuration-as-code-support:1.7")),
	},
	Must(New("kubernetes-credentials-provider:0.12.1")).String(): {
		Must(New(credentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(variantPlugin)),
	},
}

// BasePlugins returns map of plugins to install by operator
func BasePlugins() (plugins map[string][]string) {
	plugins = map[string][]string{}

	for rootPluginName, dependentPlugins := range BasePluginsMap {
		plugins[rootPluginName] = []string{}
		for _, pluginName := range dependentPlugins {
			plugins[rootPluginName] = append(plugins[rootPluginName], pluginName.String())
		}
	}

	return
}
