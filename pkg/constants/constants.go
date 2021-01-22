package constants

const (
	// OperatorName is a operator name
	OperatorName = "jenkins-operator"
	// ServiceName
	DefaultService = "jenkins-jenkins-with-all"
	DefaultJnlpService = "jenkins-jenkins-with-all-jnlp"
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
	// JavaOptsVariableName is the name of environment variable which consists Jenkins Java options
	JavaOptsVariableName = "JAVA_OPTS"
	// JavaOptsDefaultValue default value for JAVA_OPTS if not set
	JavaOptsDefaultValue = "-XX:+UnlockExperimentalVMOptions -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true -Dhudson.security.csrf.DefaultCrumbIssuer.EXCLUDE_SESSION_ID=true -Dcasc.reload.token=$(POD_NAME)"

	// KubernetesTrustCertsVariableName KUBERNETES_TRUST_CERTIFICATES var name. Tells the kubernetes plugin if it should trust the kubernetes certificate
	KubernetesTrustCertsVariableName = "KUBERNETES_TRUST_CERTIFICATES"

	// KubernetesTrustCertsDefaultValue Tells the kubernetes plugin if it should trust the kubernetes certificate
	KubernetesTrustCertsDefaultValue = "true"

	// JenkinsStatusCompleted is the completed status value.
	JenkinsStatusCompleted = "Completed"
	// JenkinsStatusReinitializing is the status given if Jenkins instance is being recreated
	JenkinsStatusReinitializing = "Reinitializing"
	// DefaultJenkinsMasterContainerName is the Jenkins master container name in pod
	DefaultJenkinsMasterContainerName = "jenkins"
	// DefaultJenkinsSideCarImage is the default jenkins sidecar image
	DefaultJenkinsSideCarImage = "quay.io/redhat-developer/jenkins-kubernetes-sidecar:0.1.144"
	// DefaultJenkinsBackupImage is the default ubi minimal image
	DefaultJenkinsBackupImage = "ubi8/ubi-minimal:latest"
)
