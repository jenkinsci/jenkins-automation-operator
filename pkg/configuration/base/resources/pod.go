package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/client/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// JenkinsMasterContainerName is the Jenkins master container name in pod
	JenkinsMasterContainerName = "jenkins-master"
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

	// GroovyScriptsSecretVolumePath is a path where are groovy scripts used to configure Jenkins
	// This script is provided by user
	//GroovyScriptsSecretVolumePath = "/tmp" + "/groovy-scripts-secrets"
	// ConfigurationAsCodeSecretVolumePath is a path where are CasC configs used to configure Jenkins
	// This script is provided by user
	ConfigurationAsCodeSecretVolumePath = "/tmp" + "/configuration-as-code-secrets"

	httpPortName = "http"
	jnlpPortName = "jnlp"
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
		{
			Name:  "SECRETS",
			Value: ConfigurationAsCodeSecretVolumePath,
		},
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

	for _, container := range jenkins.Spec.Master.Containers[1:] {
		containers = append(containers, ConvertJenkinsContainerToKubernetesContainer(container))
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
