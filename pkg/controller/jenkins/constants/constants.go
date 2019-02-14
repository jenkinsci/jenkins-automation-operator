package constants

const (
	// OperatorName is a operator name
	OperatorName = "jenkins-operator"
	// DefaultAmountOfExecutors is the default amount of Jenkins executors
	DefaultAmountOfExecutors = 3
	// SeedJobSuffix is a suffix added for all seed jobs
	SeedJobSuffix = "job-dsl-seed"
	// DefaultJenkinsMasterImage is the default Jenkins master docker image
	DefaultJenkinsMasterImage = "jenkins/jenkins:lts"
	// UserConfigurationJobName is the Jenkins job name used to configure Jenkins by groovy scripts provided by user
	UserConfigurationJobName = OperatorName + "-user-configuration"
)
