package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// JenkinsMasterContainerName is the Jenkins master container name in pod
	JenkinsMasterContainerName = "jenkins-master"
	// JenkinsHomeVolumeName is the Jenkins home volume name
	JenkinsHomeVolumeName = "jenkins-home"
	jenkinsPath           = "/var/jenkins"
	jenkinsHomePath       = jenkinsPath + "/home"

	jenkinsScriptsVolumeName = "scripts"
	// JenkinsScriptsVolumePath is a path where are scripts used to configure Jenkins
	JenkinsScriptsVolumePath = jenkinsPath + "/scripts"
	// InitScriptName is the init script name which configures init.groovy.d, scripts and install plugins
	InitScriptName = "init.sh"

	jenkinsOperatorCredentialsVolumeName = "operator-credentials"
	jenkinsOperatorCredentialsVolumePath = jenkinsPath + "/operator-credentials"

	jenkinsInitConfigurationVolumeName = "init-configuration"
	jenkinsInitConfigurationVolumePath = jenkinsPath + "/init-configuration"

	jenkinsBaseConfigurationVolumeName = "base-configuration"
	// JenkinsBaseConfigurationVolumePath is a path where are groovy scripts used to configure Jenkins
	// this scripts are provided by jenkins-operator
	JenkinsBaseConfigurationVolumePath = jenkinsPath + "/base-configuration"

	jenkinsUserConfigurationVolumeName = "user-configuration"
	// JenkinsUserConfigurationVolumePath is a path where are groovy scripts and CasC configs used to configure Jenkins
	// this script is provided by user
	JenkinsUserConfigurationVolumePath = jenkinsPath + "/user-configuration"

	userConfigurationSecretVolumeName = "user-configuration-secrets"
	// UserConfigurationSecretVolumePath is a path where are secrets used for groovy scripts and CasC configs
	UserConfigurationSecretVolumePath = jenkinsPath + "/user-configuration-secrets"

	httpPortName  = "http"
	slavePortName = "slavelistener"
	// HTTPPortInt defines Jenkins master HTTP port
	HTTPPortInt = 8080
)

func buildPodTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
}

// GetJenkinsMasterContainerBaseCommand returns default Jenkins master container command
func GetJenkinsMasterContainerBaseCommand() []string {
	return []string{
		"bash",
		"-c",
		fmt.Sprintf("%s/%s && /sbin/tini -s -- /usr/local/bin/jenkins.sh",
			JenkinsScriptsVolumePath, InitScriptName),
	}
}

// GetJenkinsMasterContainerBaseEnvs returns Jenkins master pod envs required by operator
func GetJenkinsMasterContainerBaseEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "JENKINS_HOME",
			Value: jenkinsHomePath,
		},
		{
			Name:  "JAVA_OPTS",
			Value: "-XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true",
		},
		{
			Name:  "SECRETS", // https://github.com/jenkinsci/configuration-as-code-plugin/blob/master/demos/kubernetes-secrets/README.md
			Value: UserConfigurationSecretVolumePath,
		},
	}
}

// GetJenkinsMasterPodBaseVolumes returns Jenkins master pod volumes required by operator
func GetJenkinsMasterPodBaseVolumes(jenkins *v1alpha2.Jenkins) []corev1.Volume {
	configMapVolumeSourceDefaultMode := corev1.ConfigMapVolumeSourceDefaultMode
	secretVolumeSourceDefaultMode := corev1.SecretVolumeSourceDefaultMode
	var scriptsVolumeDefaultMode int32 = 0777
	return []corev1.Volume{
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
			Name: jenkinsBaseConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &configMapVolumeSourceDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetBaseConfigurationConfigMapName(jenkins),
					},
				},
			},
		},
		{
			Name: jenkinsUserConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &configMapVolumeSourceDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetUserConfigurationConfigMapNameFromJenkins(jenkins),
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
		{
			Name: userConfigurationSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &secretVolumeSourceDefaultMode,
					SecretName:  GetUserConfigurationSecretNameFromJenkins(jenkins),
				},
			},
		},
	}
}

// GetJenkinsMasterContainerBaseVolumeMounts returns Jenkins master pod volume mounts required by operator
func GetJenkinsMasterContainerBaseVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      JenkinsHomeVolumeName,
			MountPath: jenkinsHomePath,
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
			Name:      jenkinsBaseConfigurationVolumeName,
			MountPath: JenkinsBaseConfigurationVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsUserConfigurationVolumeName,
			MountPath: JenkinsUserConfigurationVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsOperatorCredentialsVolumeName,
			MountPath: jenkinsOperatorCredentialsVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      userConfigurationSecretVolumeName,
			MountPath: UserConfigurationSecretVolumePath,
			ReadOnly:  true,
		},
	}
}

// NewJenkinsMasterContainer returns Jenkins master Kubernetes container
func NewJenkinsMasterContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	jenkinsContainer := jenkins.Spec.Master.Containers[0]
	envs := GetJenkinsMasterContainerBaseEnvs()
	envs = append(envs, jenkinsContainer.Env...)

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
				Name:          slavePortName,
				ContainerPort: constants.DefaultSlavePortInt32,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env:          envs,
		Resources:    jenkinsContainer.Resources,
		VolumeMounts: append(GetJenkinsMasterContainerBaseVolumeMounts(), jenkinsContainer.VolumeMounts...),
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

// GetJenkinsMasterPodName returns Jenkins pod name for given CR
func GetJenkinsMasterPodName(jenkins v1alpha2.Jenkins) string {
	return fmt.Sprintf("jenkins-%s", jenkins.Name)
}

// NewJenkinsMasterPod builds Jenkins Master Kubernetes Pod resource
func NewJenkinsMasterPod(objectMeta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) *corev1.Pod {

	serviceAccountName := objectMeta.Name
	objectMeta.Annotations = jenkins.Spec.Master.Annotations
	objectMeta.Name = GetJenkinsMasterPodName(*jenkins)

	return &corev1.Pod{
		TypeMeta:   buildPodTypeMeta(),
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			RestartPolicy:      corev1.RestartPolicyNever,
			SecurityContext:    jenkins.Spec.Master.SecurityContext,
			NodeSelector:       jenkins.Spec.Master.NodeSelector,
			Containers:         newContainers(jenkins),
			Volumes:            append(GetJenkinsMasterPodBaseVolumes(jenkins), jenkins.Spec.Master.Volumes...),
		},
	}
}
