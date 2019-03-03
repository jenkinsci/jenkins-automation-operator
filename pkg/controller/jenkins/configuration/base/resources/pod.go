package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	jenkinsHomeVolumeName = "home"
	jenkinsPath           = "/var/jenkins"
	jenkinsHomePath       = jenkinsPath + "/home"

	jenkinsScriptsVolumeName = "scripts"
	jenkinsScriptsVolumePath = jenkinsPath + "/scripts"
	initScriptName           = "init.sh"

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
	HTTPPortInt    = 8080
	slavePortInt   = 50000
	httpPortInt32  = int32(8080)
	slavePortInt32 = int32(50000)

	jenkinsUserUID = int64(1000) // build in Docker image jenkins user UID
)

func buildPodTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
}

// NewJenkinsMasterPod builds Jenkins Master Kubernetes Pod resource
func NewJenkinsMasterPod(objectMeta metav1.ObjectMeta, jenkins *v1alpha1.Jenkins) *corev1.Pod {
	initialDelaySeconds := int32(30)
	timeoutSeconds := int32(5)
	failureThreshold := int32(12)
	runAsUser := jenkinsUserUID

	objectMeta.Annotations = jenkins.Spec.Master.Annotations

	return &corev1.Pod{
		TypeMeta:   buildPodTypeMeta(),
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			ServiceAccountName: objectMeta.Name,
			RestartPolicy:      corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsUser,
			},
			Containers: []corev1.Container{
				{
					Name:  "jenkins-master",
					Image: jenkins.Spec.Master.Image,
					Command: []string{
						"bash",
						fmt.Sprintf("%s/%s", jenkinsScriptsVolumePath, initScriptName),
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/login",
								Port:   intstr.FromString(httpPortName),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: initialDelaySeconds,
						TimeoutSeconds:      timeoutSeconds,
						FailureThreshold:    failureThreshold,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/login",
								Port:   intstr.FromString(httpPortName),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: initialDelaySeconds,
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          slavePortName,
							ContainerPort: slavePortInt32,
						},
						{
							Name:          httpPortName,
							ContainerPort: httpPortInt32,
						},
					},
					Env: []corev1.EnvVar{
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
					},
					Resources: jenkins.Spec.Master.Resources,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      jenkinsHomeVolumeName,
							MountPath: jenkinsHomePath,
							ReadOnly:  false,
						},
						{
							Name:      jenkinsScriptsVolumeName,
							MountPath: jenkinsScriptsVolumePath,
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
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: jenkinsHomeVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: jenkinsScriptsVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
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
							SecretName: GetOperatorCredentialsSecretName(jenkins),
						},
					},
				},
				{
					Name: userConfigurationSecretVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: GetUserConfigurationSecretNameFromJenkins(jenkins),
						},
					},
				},
			},
		},
	}
}
