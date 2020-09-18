package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/client/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	JenkinsMasterContainerName = constants.DefaultJenkinsMasterContainerName
	// JenkinsHomeVolumeName is the Jenkins home volume name
	JenkinsHomeVolumeName = "jenkins-home"
	jenkinsPath           = "/var/jenkins"

	jenkinsScriptsVolumeName = "scripts"
	// JenkinsScriptsVolumePath is a path where are scripts used to configure Jenkins
	JenkinsScriptsVolumePath = jenkinsPath + "/scripts"
	// InitScriptName is the init script name which configures init.groovy.d, scripts and install plugins
	InitScriptName = "init.sh"

	jenkinsOperatorCredentialsVolumeName = "operator-credentials"
	jenkinsOperatorCredentialsVolumePath = jenkinsPath + "/operator-credentials"

	jenkinsInitConfigurationVolumeName = "init-configuration"
	jenkinsInitConfigurationVolumePath = jenkinsPath + "/init-configuration"

	ConfigurationAsCodeVolumeName       = "casc-config"
	ConfigurationAsCodeVolumePath       = jenkinsPath + "/configuration-as-code"
	ConfigurationAsCodeSecretVolumeName = "casc-secret"
	ConfigurationAsCodeSecretVolumePath = "/tmp" + "/configuration-as-code-secrets"

	httpPortName = "http"
	jnlpPortName = "jnlp"

	// JenkinsSCConfigName is the Jenkins side car container name for reloading config
	JenkinsSCConfigName            = "jenkins-sc-config"
	JenkinsSCConfigImage           = "kiwigrid/k8s-sidecar:0.1.144"
	JenkinsSCConfigImagePullPolicy = "IfNotPresent"
	JenkinsSCConfigReqURL          = "http://localhost:8080/reload-configuration-as-code/?casc-reload-token=$(POD_NAME)"
	JenkinsSCConfigReqMethod       = "POST"
	JenkinsSCConfigReqRetry        = "10"
	JenkinsSCConfigCPULimit        = "100m"
	JenkinsSCConfigMEMLimit        = "100Mi"
	JenkinsSCConfigCPURequest      = "50m"
	JenkinsSCConfigMEMRequest      = "50Mi"
	JenkinsSCConfigLabel           = "type"
	JenkinsSCConfigLabelValue      = "%s-jenkins-config"
)

// GetJenkinsMasterContainerBaseCommand returns default Jenkins master container command
func GetJenkinsMasterContainerBaseCommand() []string {
	return []string{
		"bash",
		"-c",
		fmt.Sprintf("%s/%s && exec /sbin/tini -s -- /usr/local/bin/jenkins.sh",
			JenkinsScriptsVolumePath, InitScriptName),
	}
}

// GetJenkinsMasterContainerBaseEnvs returns Jenkins master pod envs required by operator
func GetJenkinsMasterContainerBaseEnvs(jenkins *v1alpha2.Jenkins) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "OPENSHIFT_ENABLE_OAUTH",
			Value: "true",
		},
		{
			Name:  "OPENSHIFT_ENABLE_REDIRECT_PROMPT",
			Value: "true",
		},
		{
			Name:  "COPY_REFERENCE_FILE_LOG",
			Value: fmt.Sprintf("%s/%s", getJenkinsHomePath(jenkins), "copy_reference_file.log"),
		},
	}

	if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SECRETS",
			Value: ConfigurationAsCodeSecretVolumePath,
		})
	}

	if jenkins.Spec.ConfigurationAsCode.Enabled {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "CASC_JENKINS_CONFIG",
			Value: ConfigurationAsCodeVolumePath,
		})
	}

	return envVars
}

// getJenkinsHomePath fetches the Home Path for Jenkins
func getJenkinsHomePath(jenkins *v1alpha2.Jenkins) string {
	defaultJenkinsHomePath := "/var/lib/jenkins"
	for _, envVar := range jenkins.Spec.Master.Containers[0].Env {
		if envVar.Name == "JENKINS_HOME" {
			return envVar.Value
		}
	}
	return defaultJenkinsHomePath
}

// GetJenkinsMasterPodBaseVolumes returns Jenkins master pod volumes required by operator
func GetJenkinsMasterPodBaseVolumes(jenkins *v1alpha2.Jenkins) []corev1.Volume {
	configMapVolumeSourceDefaultMode := corev1.ConfigMapVolumeSourceDefaultMode
	secretVolumeSourceDefaultMode := corev1.SecretVolumeSourceDefaultMode
	var scriptsVolumeDefaultMode int32 = 0777
	volumes := []corev1.Volume{
		{
			Name: JenkinsHomeVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: jenkinsScriptsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getScriptsConfigMapName(jenkins),
					},
				},
			},
		},
		{
			Name: jenkinsInitConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &configMapVolumeSourceDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetInitConfigurationConfigMapName(jenkins),
					},
				},
			},
		},
		{
			Name: jenkinsOperatorCredentialsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &secretVolumeSourceDefaultMode,
					SecretName:  GetOperatorCredentialsSecretName(jenkins),
				},
			},
		},
	}

	if jenkins.Spec.ConfigurationAsCode.Enabled {
		// target volume for the init container
		// All casc configmaps will be copied here
		volumes = append(volumes, corev1.Volume{
			Name: ConfigurationAsCodeVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		// Loop to add all casc configmap volumes
		for _, cm := range jenkins.Spec.ConfigurationAsCode.Configurations {
			volumes = append(volumes, corev1.Volume{
				Name: fmt.Sprintf("casc-init-%s", cm.Name),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						DefaultMode: &configMapVolumeSourceDefaultMode,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cm.Name,
						},
					},
				},
			})
		}
		// Add casc secret volume
		if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
			volumes = append(volumes, corev1.Volume{
				Name: ConfigurationAsCodeSecretVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &secretVolumeSourceDefaultMode,
						SecretName:  jenkins.Spec.ConfigurationAsCode.Secret.Name,
					},
				},
			})
		}
	}

	return volumes
}

// GetJenkinsMasterContainerBaseVolumeMounts returns Jenkins master pod volume mounts required by operator
func GetJenkinsMasterContainerBaseVolumeMounts(jenkins *v1alpha2.Jenkins) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      JenkinsHomeVolumeName,
			MountPath: getJenkinsHomePath(jenkins),
			ReadOnly:  false,
		},
		{
			Name:      jenkinsScriptsVolumeName,
			MountPath: JenkinsScriptsVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsInitConfigurationVolumeName,
			MountPath: jenkinsInitConfigurationVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsOperatorCredentialsVolumeName,
			MountPath: jenkinsOperatorCredentialsVolumePath,
			ReadOnly:  true,
		},
	}

	if jenkins.Spec.ConfigurationAsCode.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      ConfigurationAsCodeVolumeName,
			MountPath: ConfigurationAsCodeVolumePath,
			ReadOnly:  false,
		})
		if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      ConfigurationAsCodeSecretVolumeName,
				MountPath: ConfigurationAsCodeSecretVolumePath,
				ReadOnly:  false,
			})
		}
	}

	return volumeMounts
}

// NewJenkinsMasterContainer returns Jenkins master Kubernetes container
func NewJenkinsMasterContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	jenkinsContainer := jenkins.Spec.Master.Containers[0]

	envs := GetJenkinsMasterContainerBaseEnvs(jenkins)
	envs = append(envs, jenkinsContainer.Env...)

	jenkinsHomeEnvVar := corev1.EnvVar{
		Name:  "JENKINS_HOME",
		Value: getJenkinsHomePath(jenkins),
	}

	jenkinsHomeEnvVarExists := false
	for _, env := range jenkinsContainer.Env {
		if env.Name == jenkinsHomeEnvVar.Name {
			jenkinsHomeEnvVarExists = true
			break
		}
	}

	if !jenkinsHomeEnvVarExists {
		envs = append(envs, jenkinsHomeEnvVar)
	}

	return corev1.Container{
		Name:            JenkinsMasterContainerName,
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         jenkinsContainer.Command,
		LivenessProbe:   jenkinsContainer.LivenessProbe,
		ReadinessProbe:  jenkinsContainer.ReadinessProbe,
		Ports: []corev1.ContainerPort{
			{
				Name:          httpPortName,
				ContainerPort: constants.DefaultHTTPPortInt32,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          jnlpPortName,
				ContainerPort: constants.DefaultJNLPPortInt32,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		SecurityContext: jenkinsContainer.SecurityContext,
		Env:             envs,
		Resources:       jenkinsContainer.Resources,
		VolumeMounts:    append(GetJenkinsMasterContainerBaseVolumeMounts(jenkins), jenkinsContainer.VolumeMounts...),
	}
}

// NewJenkinsSCConfigContainer returns Jenkins side container for config reloading
func NewJenkinsSCConfigContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	envs := map[string]string{
		"LABEL":             JenkinsSCConfigLabel,
		"LABEL_VALUE":       fmt.Sprintf(JenkinsSCConfigLabelValue, jenkins.Name),
		"FOLDER":            ConfigurationAsCodeVolumePath,
		"REQ_URL":           JenkinsSCConfigReqURL,
		"REQ_METHOD":        JenkinsSCConfigReqMethod,
		"REQ_RETRY_CONNECT": JenkinsSCConfigReqRetry,
	}

	envVars := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	for k, v := range envs {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      ConfigurationAsCodeVolumeName,
			MountPath: ConfigurationAsCodeVolumePath,
			ReadOnly:  false,
		},
		{
			Name:      "jenkins-home",
			MountPath: getJenkinsHomePath(jenkins),
			ReadOnly:  true,
		},
	}

	return corev1.Container{
		Name:            JenkinsSCConfigName,
		Image:           JenkinsSCConfigImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(JenkinsSCConfigCPURequest),
				corev1.ResourceMemory: resource.MustParse(JenkinsSCConfigMEMRequest),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(JenkinsSCConfigCPULimit),
				corev1.ResourceMemory: resource.MustParse(JenkinsSCConfigMEMLimit),
			},
		},
		VolumeMounts: volumeMounts,
	}
}

// NewJenkinsInitContainer returns Jenkins init container to copy configmap to make it writable
func NewJenkinsInitContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	jenkinsContainer := jenkins.Spec.Master.Containers[0]
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      ConfigurationAsCodeVolumeName,
			MountPath: ConfigurationAsCodeVolumePath,
			ReadOnly:  false,
		},
	}

	if jenkins.Spec.ConfigurationAsCode.Enabled {
		for _, cm := range jenkins.Spec.ConfigurationAsCode.Configurations {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      fmt.Sprintf("casc-init-%s", cm.Name),
				MountPath: jenkinsPath + fmt.Sprintf("/casc-init-%s", cm.Name),
				ReadOnly:  false,
			})
		}
	}

	command := []string{
		"bash",
		"-c",
		fmt.Sprintf("find %s/casc-init* -type f |xargs cp -rfLt %s", jenkinsPath, ConfigurationAsCodeVolumePath),
	}
	return corev1.Container{
		Name:            fmt.Sprintf("init-%s", jenkinsContainer.Name),
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         command,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(JenkinsSCConfigCPURequest),
				corev1.ResourceMemory: resource.MustParse(JenkinsSCConfigMEMRequest),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(JenkinsSCConfigCPULimit),
				corev1.ResourceMemory: resource.MustParse(JenkinsSCConfigMEMLimit),
			},
		},
		VolumeMounts: volumeMounts,
	}
}

// ConvertJenkinsContainerToKubernetesContainer converts Jenkins container to Kubernetes container
func ConvertJenkinsContainerToKubernetesContainer(container v1alpha2.Container) corev1.Container {
	return corev1.Container{
		Name:            container.Name,
		Image:           container.Image,
		Command:         container.Command,
		Args:            container.Args,
		WorkingDir:      container.WorkingDir,
		Ports:           container.Ports,
		EnvFrom:         container.EnvFrom,
		Env:             container.Env,
		Resources:       container.Resources,
		VolumeMounts:    container.VolumeMounts,
		LivenessProbe:   container.LivenessProbe,
		ReadinessProbe:  container.ReadinessProbe,
		Lifecycle:       container.Lifecycle,
		ImagePullPolicy: container.ImagePullPolicy,
		SecurityContext: container.SecurityContext,
	}
}

func newContainers(jenkins *v1alpha2.Jenkins) (containers []corev1.Container) {
	containers = append(containers, NewJenkinsMasterContainer(jenkins))
	if jenkins.Spec.ConfigurationAsCode.Enabled && jenkins.Spec.ConfigurationAsCode.EnableAutoReload  {
		containers = append(containers, NewJenkinsSCConfigContainer(jenkins))
	}
	for _, container := range jenkins.Spec.Master.Containers[1:] {
		containers = append(containers, ConvertJenkinsContainerToKubernetesContainer(container))
	}

	return
}

func newInitContainers(jenkins *v1alpha2.Jenkins) (containers []corev1.Container) {
	if jenkins.Spec.ConfigurationAsCode.Enabled {
		containers = append(containers, NewJenkinsInitContainer(jenkins))
	}
	return
}

// GetJenkinsMasterPodLabels returns Jenkins pod labels for given CR
func GetJenkinsMasterPodLabels(jenkins *v1alpha2.Jenkins) map[string]string {
	var labels map[string]string
	if jenkins.Spec.Master.Labels == nil {
		labels = map[string]string{}
	} else {
		labels = jenkins.Spec.Master.Labels
	}
	for key, value := range BuildResourceLabels(jenkins) {
		labels[key] = value
	}
	return labels
}

// return a condition function that indicates whether the given pod is
// currently running
func isPodRunning(k8sClient client.Client, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod := &corev1.Pod{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: namespace}, pod)
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return false, conditions.ErrPodCompleted
		}
		return false, nil
	}
}

// Poll up to timeout seconds for pod to enter running state.
// Returns an error if the pod never enters the running state.
func WaitForPodRunning(k8sClient client.Client, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isPodRunning(k8sClient, podName, namespace))
}
