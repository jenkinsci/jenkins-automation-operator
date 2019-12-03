package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1.34"
	configurationAsCodeSupportPlugin    = "configuration-as-code-support:1.19"
	gitPlugin                           = "git:4.0.0"
	jobDslPlugin                        = "job-dsl:1.76"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:0.13"
	kubernetesPlugin                    = "kubernetes:1.21.3"
	workflowAggregatorPlugin            = "workflow-aggregator:2.6"
	workflowJobPlugin                   = "workflow-job:2.36"
)

// basePluginsList contains plugins to install by operator
var basePluginsList = []Plugin{
	Must(New(kubernetesPlugin)),
	Must(New(workflowJobPlugin)),
	Must(New(workflowAggregatorPlugin)),
	Must(New(gitPlugin)),
	Must(New(jobDslPlugin)),
	Must(New(configurationAsCodePlugin)),
	Must(New(configurationAsCodeSupportPlugin)),
	Must(New(kubernetesCredentialsProviderPlugin)),
}

// BasePlugins returns list of plugins to install by operator
func BasePlugins() []Plugin {
	return basePluginsList
}
