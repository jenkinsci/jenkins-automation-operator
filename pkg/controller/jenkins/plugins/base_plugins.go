package plugins

const (
	aceEditorPlugin                     = "ace-editor:1.1"
	apacheComponentsClientPlugin        = "apache-httpcomponents-client-4-api:4.5.5-3.0"
	authenticationTokensPlugin          = "authentication-tokens:1.3"
	branchApiPlugin                     = "branch-api:2.5.2"
	cloudBeesFolderPlugin               = "cloudbees-folder:6.8"
	configurationAsCodePlugin           = "configuration-as-code:1.17"
	configurationAsCodeSupportPlugin    = "configuration-as-code-support:1.17"
	credentialsBindingPlugin            = "credentials-binding:1.18"
	credentialsPlugin                   = "credentials:2.1.19"
	displayURLAPIPlugin                 = "display-url-api:2.3.1"
	dockerCommonsPlugin                 = "docker-commons:1.15"
	dockerWorkflowPlugin                = "docker-workflow:1.18"
	durableTaskPlugin                   = "durable-task:1.29"
	gitClientPlugin                     = "git-client:2.7.7"
	gitPlugin                           = "git:3.10.0"
	gitServerPlugin                     = "git-server:1.7"
	handlebarsPlugin                    = "handlebars:1.1.1"
	jackson2ADIPlugin                   = "jackson2-api:2.9.9"
	jobDslPlugin                        = "job-dsl:1.74"
	jqueryDetachedPlugin                = "jquery-detached:1.2.1"
	jschPlugin                          = "jsch:0.1.55"
	junitPlugin                         = "junit:1.28"
	kubernetesCredentialsPlugin         = "kubernetes-credentials:0.4.0"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:0.12.1"
	kubernetesPlugin                    = "kubernetes:1.15.5"
	lockableResourcesPlugin             = "lockable-resources:2.5"
	mailerPlugin                        = "mailer:1.23"
	matrixProjectPlugin                 = "matrix-project:1.14"
	momentjsPlugin                      = "momentjs:1.1.1"
	pipelineBuildStepPlugin             = "pipeline-build-step:2.9"
	pipelineGraphAnalysisPlugin         = "pipeline-graph-analysis:1.10"
	pipelineInputStepPlugin             = "pipeline-input-step:2.10"
	pipelineMilestoneStepPlugin         = "pipeline-milestone-step:1.3.1"
	pipelineModelApiPlugin              = "pipeline-model-api:1.3.8"
	pipelineModelDeclarativeAgentPlugin = "pipeline-model-declarative-agent:1.1.1"
	pipelineModelDefinitionPlugin       = "pipeline-model-definition:1.3.8"
	pipelineModelExtensionsPlugin       = "pipeline-model-extensions:1.3.8"
	pipelineRestApiPlugin               = "pipeline-rest-api:2.11"
	pipelineStageStepPlugin             = "pipeline-stage-step:2.3"
	pipelineStageTagsMetadataPlugin     = "pipeline-stage-tags-metadata:1.3.8"
	pipelineStageViewPlugin             = "pipeline-stage-view:2.11"
	plainCredentialsPlugin              = "plain-credentials:1.5"
	scmAPIPlugin                        = "scm-api:2.4.1"
	scriptSecurityPlugin                = "script-security:1.59"
	sshCredentialsPlugin                = "ssh-credentials:1.16"
	structsPlugin                       = "structs:1.19"
	variantPlugin                       = "variant:1.2"
	workflowAggregatorPlugin            = "workflow-aggregator:2.6"
	workflowAPIPlugin                   = "workflow-api:2.34"
	workflowBasicStepsPlugin            = "workflow-basic-steps:2.16"
	workflowCpsGlobalLibPlugin          = "workflow-cps-global-lib:2.13"
	workflowCpsPlugin                   = "workflow-cps:2.69"
	workflowDurableTaskStepPlugin       = "workflow-durable-task-step:2.30"
	workflowJobPlugin                   = "workflow-job:2.32"
	workflowMultibranchPlugin           = "workflow-multibranch:2.21"
	workflowSCMStepPlugin               = "workflow-scm-step:2.7"
	workflowStepAPIPlugin               = "workflow-step-api:2.19"
	workflowSupportPlugin               = "workflow-support:3.3"
)

// BasePluginsMap contains plugins to install by operator
var BasePluginsMap = map[string][]Plugin{
	Must(New(kubernetesPlugin)).String(): {
		Must(New(apacheComponentsClientPlugin)),
		Must(New(cloudBeesFolderPlugin)),
		Must(New(credentialsPlugin)),
		Must(New(durableTaskPlugin)),
		Must(New(jackson2ADIPlugin)),
		Must(New(kubernetesCredentialsPlugin)),
		Must(New(plainCredentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(variantPlugin)),
		Must(New(workflowStepAPIPlugin)),
	},
	Must(New(workflowJobPlugin)).String(): {
		Must(New(scmAPIPlugin)),
		Must(New(scriptSecurityPlugin)),
		Must(New(structsPlugin)),
		Must(New(workflowAPIPlugin)),
		Must(New(workflowStepAPIPlugin)),
		Must(New(workflowSupportPlugin)),
	},
	Must(New(workflowAggregatorPlugin)).String(): {
		Must(New(aceEditorPlugin)),
		Must(New(apacheComponentsClientPlugin)),
		Must(New(authenticationTokensPlugin)),
		Must(New(branchApiPlugin)),
		Must(New(cloudBeesFolderPlugin)),
		Must(New(credentialsBindingPlugin)),
		Must(New(credentialsPlugin)),
		Must(New(displayURLAPIPlugin)),
		Must(New(dockerCommonsPlugin)),
		Must(New(dockerWorkflowPlugin)),
		Must(New(durableTaskPlugin)),
		Must(New(gitClientPlugin)),
		Must(New(gitServerPlugin)),
		Must(New(handlebarsPlugin)),
		Must(New(jackson2ADIPlugin)),
		Must(New(jqueryDetachedPlugin)),
		Must(New(jschPlugin)),
		Must(New(junitPlugin)),
		Must(New(lockableResourcesPlugin)),
		Must(New(mailerPlugin)),
		Must(New(matrixProjectPlugin)),
		Must(New(momentjsPlugin)),
		Must(New(pipelineBuildStepPlugin)),
		Must(New(pipelineGraphAnalysisPlugin)),
		Must(New(pipelineInputStepPlugin)),
		Must(New(pipelineMilestoneStepPlugin)),
		Must(New(pipelineModelApiPlugin)),
		Must(New(pipelineModelDeclarativeAgentPlugin)),
		Must(New(pipelineModelDefinitionPlugin)),
		Must(New(pipelineModelExtensionsPlugin)),
		Must(New(pipelineRestApiPlugin)),
		Must(New(pipelineStageStepPlugin)),
		Must(New(pipelineStageTagsMetadataPlugin)),
		Must(New(pipelineStageViewPlugin)),
		Must(New(plainCredentialsPlugin)),
		Must(New(scmAPIPlugin)),
		Must(New(scriptSecurityPlugin)),
		Must(New(sshCredentialsPlugin)),
		Must(New(structsPlugin)),
		Must(New(workflowAPIPlugin)),
		Must(New(workflowBasicStepsPlugin)),
		Must(New(workflowCpsGlobalLibPlugin)),
		Must(New(workflowCpsPlugin)),
		Must(New(workflowDurableTaskStepPlugin)),
		Must(New(workflowJobPlugin)),
		Must(New(workflowMultibranchPlugin)),
		Must(New(workflowSCMStepPlugin)),
		Must(New(workflowStepAPIPlugin)),
		Must(New(workflowSupportPlugin)),
	},
	Must(New(gitPlugin)).String(): {
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
	Must(New(jobDslPlugin)).String(): {
		Must(New(scriptSecurityPlugin)),
		Must(New(structsPlugin)),
	},
	Must(New(configurationAsCodePlugin)).String(): {
		Must(New(configurationAsCodeSupportPlugin)),
	},
	Must(New(kubernetesCredentialsProviderPlugin)).String(): {
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
