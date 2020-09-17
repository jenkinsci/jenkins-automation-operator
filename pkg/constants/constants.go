package constants

const (
	// OperatorName is a operator name
	OperatorName = "jenkins-operator"
	// DefaultAmountOfExecutors is the default amount of Jenkins executors
	DefaultAmountOfExecutors = 0
	// SeedJobSuffix is a suffix added for all seed jobs
	SeedJobSuffix = "job-dsl-seed"
	// DefaultJenkinsMasterImage is the default Jenkins master docker image
	DefaultJenkinsMasterImage = "jenkins/jenkins:lts"
	// DefaultJenkinsMasterImage is the default Jenkins master docker image
	DefaultOpenShiftJenkinsMasterImage = "registry.redhat.io/openshift4/ose-jenkins:v4.5"
	// DefaultHTTPPortInt32 is the default Jenkins HTTP port
	DefaultHTTPPortInt32 = int32(8080)
	// DefaultJNLPPortInt32 is the default Jenkins port for slaves
	DefaultJNLPPortInt32 = int32(50000)
	// JavaOpsVariableName is the name of environment variable which consists Jenkins Java options
	JavaOpsVariableName = "JAVA_OPTS"
	// JenkinsStatusCompleted is the completed status value.
	JenkinsStatusCompleted = "Completed"
	// JenkinsStatusReinitializing is the status given if Jenkins instance is being recreated
	JenkinsStatusReinitializing = "Reinitializing"
)
