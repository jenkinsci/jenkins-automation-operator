package v1alpha2

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsSpec defines the desired state of the Jenkins.
// +k8s:openapi-gen=true
// +operator-sdk:csv:customresourcedefinitions:displayName="Jenkins"
type JenkinsSpec struct {
	// Master represents Jenkins master pod properties and Jenkins plugins.
	// Every single change here requires a pod restart.
	Master *JenkinsMaster `json:"master,omitempty"`

	// WIP: Still need to determine if  a reference to JenkinsImage is sufficient or if we need a
	// JenkinsImageBuild object to point to a specific build.
	// If JenkinsImageRef is set, we need delete the pod to re-trigger a build

	// JenkinsImageRef a reference to a JenkinsImage in the current namespace. The JenkinsImage must have
	// status to be "SuccessfullyBuilt" ; then the image target image in JenkinsImage.To will be used
	// as the image of the Master container.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="JenkinsImage reference",xDescriptors="urn:alm:descriptor:io.jenkins:JenkinsImage"
	JenkinsImageRef string `json:"jenkinsImageRef,omitempty"`

	// ForceBasePluginsInstall forces the installation of the minimum required basePlugins during Jenkins container startup
	// in a postStart lifecycle.
	// Defaults to : false
	// +optional

	ForceBasePluginsInstall bool `json:"forceBasePluginsInstall,omitempty"`

	// Service is Kubernetes service of Jenkins master HTTP pod
	// Defaults to :
	// port: 8080
	// type: ClusterIP
	// +optional
	Service Service `json:"service,omitempty"`

	// Service is Kubernetes service of Jenkins agent pods
	// Defaults to :
	// port: 50000
	// type: ClusterIP
	// +optional
	JNLPService Service `json:"jnlpService,omitempty"`

	// Roles defines list of extra RBAC roles for the Jenkins Master pod service account
	// +optional
	Roles []rbacv1.RoleRef `json:"roles,omitempty"`

	// ServiceAccount defines Jenkins master service account attributes
	// +optional
	ServiceAccount ServiceAccount `json:"serviceAccount,omitempty"`

	// JenkinsAPISettings defines configuration used by the operator to gain admin access to the Jenkins API
	JenkinsAPISettings JenkinsAPISettings `json:"jenkinsAPISettings,omitempty"`

	// ConfigurationAsCode defines configuration of Jenkins configuration via Configuration as Code Jenkins plugin
	// +optional
	ConfigurationAsCode *Configuration `json:"configurationAsCode,omitempty"`

	// BackupEnabled defines whether backup feature is enabled
	BackupEnabled bool `json:"backupEnabled,omitempty"`

	// MetricsEnabled defines whether prometheus metrics are enabled
	MetricsEnabled bool `json:"metricsEnabled,omitempty"`
}

// AuthorizationStrategy defines authorization strategy of the operator for the Jenkins API
type AuthorizationStrategy string

const (
	// CreateUserAuthorizationStrategy operator sets HudsonPrivateSecurityRealm and FullControlOnceLoggedInAuthorizationStrategy than creates user using init.d groovy script
	CreateUserAuthorizationStrategy AuthorizationStrategy = "createUser"
	// ServiceAccountAuthorizationStrategy operator gets token associated with Jenkins service account and uses it as bearer token
	ServiceAccountAuthorizationStrategy AuthorizationStrategy = "serviceAccount"
)

// JenkinsAPISettings defines configuration used by the operator to gain admin access to the Jenkins API
type JenkinsAPISettings struct {
	AuthorizationStrategy AuthorizationStrategy `json:"authorizationStrategy"`
}

// SecretRef is reference to Kubernetes secret
type SecretRef struct {
	Name string `json:"name"`
}

// ConfigMapRef is reference to Kubernetes ConfigMap
type ConfigMapRef struct {
	Name string `json:"name"`
}

// Configuration defines a Jenkins Configuration
type Configuration struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable this configuration",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	Enabled          bool           `json:"enabled"`
	DefaultConfig    bool           `json:"defaultConfig"`
	Secret           SecretRef      `json:"secret,omitempty"`
	Configurations   []ConfigMapRef `json:"configurations,omitempty"`
	EnableAutoReload bool           `json:"enableAutoReload"`
}

// ServiceAccount defines Kubernetes service account attributes
type ServiceAccount struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NotificationLevel defines the level of a Notification.
type NotificationLevel string

const (
	// NotificationLevelWarning - Only Warnings
	NotificationLevelWarning NotificationLevel = "warning"

	// NotificationLevelInfo - Only info
	NotificationLevelInfo NotificationLevel = "info"
)

// SecretKeySelector selects a key of a Secret.
type SecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	corev1.LocalObjectReference `json:"secret"`
	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
}

// Container defines Kubernetes container attributes.
type Container struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	Name string `json:"name"`

	// Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Compute Resources required by this container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty"`

	// Container's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`

	// List of ports to expose from the container. Exposing a port here gives
	// the system additional information about the network connections a
	// container uses, but is primarily informational. Not specifying a port here
	// DOES NOT prevent that port from being exposed. Any port which is
	// listening on the default "0.0.0.0" address inside a container will be
	// accessible from the network.
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// List of environment variables to set in the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Pod volumes to mount into the container's filesystem.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Actions that the management system should take in response to container lifecycle events.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`

	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

// Plugin defines Jenkins plugin.
type Plugin struct {
	// Name is the name of Jenkins plugin
	Name string `json:"name"`
	// Version is the version of Jenkins plugin
	Version string `json:"version"`
	// DownloadURL is the custom url from where plugin has to be downloaded.
	DownloadURL string `json:"downloadURL,omitempty"`
}

// JenkinsMaster defines the Jenkins master pod attributes and plugins,
// every single change requires a Jenkins master pod restart.
type JenkinsMaster struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// Deprecated: will be removed in the future, please use Annotations(annotations)
	// +optional
	AnnotationsDeprecated map[string]string `json:"masterAnnotations,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// SecurityContext that applies to all the containers of the Jenkins
	// Master. As per kubernetes specification, it can be overridden
	// for each container individually.
	// +optional
	// Defaults to:
	// runAsUser: 1000
	// fsGroup: 1000
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// List of containers belonging to the pod.
	// Containers cannot currently be added or removed.
	// There must be at least one container in a Pod.
	// +optional
	// Defaults to:
	// - image: jenkins/jenkins:lts
	//   imagePullPolicy: Always
	//   livenessProbe:
	//     failureThreshold: 12
	//     httpGet:
	//       path: /login
	//       port: http
	//       scheme: HTTP
	//     initialDelaySeconds: 80
	//     periodSeconds: 10
	//     successThreshold: 1
	//     timeoutSeconds: 5
	//   name: jenkins-master
	//   readinessProbe:
	//     failureThreshold: 3
	//     httpGet:
	//       path: /login
	//       port: http
	//       scheme: HTTP
	//     initialDelaySeconds: 30
	//     periodSeconds: 10
	//     successThreshold: 1
	//     timeoutSeconds: 1
	//   resources:
	//     limits:
	//       cpu: 1500m
	//       memory: 3Gi
	//     requests:
	//       cpu: "1"
	//       memory: 600Mi
	Containers []Container `json:"containers,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// BasePlugins contains plugins required by operator
	// +optional
	// Defaults to :
	// - name: kubernetes
	// version: 1.15.7
	// - name: workflow-job
	// version: "2.39"
	// - name: workflow-aggregator
	// version: "2.6"
	// - name: git
	// version: 3.10.0
	// - name: job-dsl
	// version: "1.74"
	// - name: configuration-as-code
	// version: "1.19"
	// - name: kubernetes-credentials-provider
	// version: 0.12.1
	BasePlugins []Plugin `json:"basePlugins,omitempty"`

	// PriorityClassName for Jenkins master pod
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// Service defines Kubernetes service attributes
type Service struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Route service traffic to pods with label keys and values matching this
	// selector. If empty or not present, the service is assumed to have an
	// external process managing its endpoints, which Kubernetes will not
	// modify. Only applies to types ClusterIP, NodePort, and LoadBalancer.
	// Ignored if type is ExternalName.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Type determines how the Service is exposed. Defaults to ClusterIP. Valid
	// options are ExternalName, ClusterIP, NodePort, and LoadBalancer.
	// "ExternalName" maps to the specified externalName.
	// "ClusterIP" allocates a cluster-internal IP address for load-balancing to
	// endpoints. Endpoints are determined by the selector or if that is not
	// specified, by manual construction of an Endpoints object. If clusterIP is
	// "None", no virtual IP is allocated and the endpoints are published as a
	// set of endpoints rather than a stable IP.
	// "NodePort" builds on ClusterIP and allocates a port on every node which
	// routes to the clusterIP.
	// "LoadBalancer" builds on NodePort and creates an
	// external load-balancer (if supported in the current cloud) which routes
	// to the clusterIP.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// The port that are exposed by this service.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	Port int32 `json:"port,omitempty"`

	// The PortName that are exposed by this service.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	PortName string `json:"portName,omitempty"`

	// The port on each node on which this service is exposed when type=NodePort or LoadBalancer.
	// Usually assigned by the system. If specified, it will be allocated to the service
	// if unused or else creation of the service will fail.
	// Default is to auto-allocate a port if the ServiceType of this Service requires one.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`

	// If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs. This field will be ignored if the
	// cloud-provider does not support the feature."
	// More info: https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`

	// Only applies to Service Type: LoadBalancer
	// LoadBalancer will get created with the IP specified in this field.
	// This feature depends on whether the underlying cloud-provider supports specifying
	// the loadBalancerIP when a load balancer is created.
	// This field will be ignored if the cloud-provider does not support the feature.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
// +operator-sdk:csv:customresourcedefinitions.type=status

type JenkinsStatus struct {
	// OperatorVersion is the operator version which manages this CR
	// +optional
	OperatorVersion string `json:"operatorVersion,omitempty"`

	// Conditions describes the state of the jenkins resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// ProvisionStartTime is a time when Jenkins master pod has been created
	// +optional
	ProvisionStartTime *metav1.Time `json:"provisionStartTime,omitempty"`

	// UserAndPasswordHash is a SHA256 hash made from user and password
	// +optional
	UserAndPasswordHash string `json:"userAndPasswordHash,omitempty"`

	// Spec defines the effective state of the Jenkins
	Spec *JenkinsSpec `json:"spec,omitempty"`
}

// +genclient

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Jenkins
	Spec JenkinsSpec `json:"spec,omitempty"`

	// Status defines the observed state of Jenkins
	Status *JenkinsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JenkinsList contains a list of Jenkins.
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}

func (jenkins *Jenkins) GetNamespace() string {
	return jenkins.ObjectMeta.Namespace
}

//nolint: stylecheck
func (jenkins *Jenkins) GetCRName() string {
	return jenkins.ObjectMeta.Name
}

func init() {
	SchemeBuilder.Register(&Jenkins{}, &JenkinsList{})
}
