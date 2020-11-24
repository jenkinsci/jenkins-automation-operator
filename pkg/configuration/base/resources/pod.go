package resources

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/client/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	JenkinsSideCarImageEnvVar = "JENKINS_SIDECAR_IMAGE"
	JenkinsBackupImageEnvVar  = "JENKINS_BACKUP_IMAGE"

	JenkinsMasterContainerName = constants.DefaultJenkinsMasterContainerName
	// JenkinsHomeVolumeName is the Jenkins home volume name
	JenkinsHomeVolumeName = "jenkins-home"
	jenkinsPath           = "/var/jenkins"

	jenkinsScriptsVolumeName = "scripts"
	// JenkinsScriptsVolumePath is a path where are scripts used to configure Jenkins
	JenkinsScriptsVolumePath = jenkinsPath + "/scripts"
	// InitScriptName is the init script name which configures init.groovy.d, scripts and install plugins
	InitScriptName = "init.sh"

	basePluginsVolumeName = "base-plugins"
	basePluginsFileName   = "base-plugins"
	// BasePluginsVolumePath is a path where the base-plugins file is generated
	BasePluginsVolumePath = jenkinsPath + "/" + basePluginsFileName

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

	// defaut configmap for jenkins configuration
	JenkinsDefaultConfigMapName = "jenkins-default-configuration"

	// Names of Sidecar and Init Containers
	ConfigSidecarName        = "config"
	ConfigInitContainerName  = "config-init"
	PluginsInitContainerName = "plugins-init"
	BackupSidecarName        = "backup"
	BackupInitContainerName  = "backup-init"
	// Config Sidecar related variables
	JenkinsSCConfigReqURL     = "http://localhost:8080/reload-configuration-as-code/?casc-reload-token=$(POD_NAME)"
	JenkinsSCConfigReqMethod  = "POST"
	JenkinsSCConfigReqRetry   = "10"
	JenkinsSCConfigLabel      = "type"
	JenkinsSCConfigLabelValue = "%s-jenkins-config"
	// Backup Sidecar related variables
	JenkinsBackupVolumeMountName = "backup-pool"
	JenkinsBackupVolumePath      = "/jenkins-backups"
	// Helper scripts related variables
	ScriptsVolumeMountName    = "helper-scripts"
	ScriptsVolumePath         = "/jenkins-operator-scripts"
	QuietDownScriptPath       = ScriptsVolumePath + "/quietdown.sh"
	CancelQuietDownScriptPath = ScriptsVolumePath + "/cancelquietdown.sh"
	RestartScriptPath         = ScriptsVolumePath + "/restart.sh"
	SafeRestartScriptPath     = ScriptsVolumePath + "/saferestart.sh"
	// Common attributes used for Sidecars
	SidecarCPULimit   = "100m"
	SidecarMEMLimit   = "100Mi"
	SidecarCPURequest = "50m"
	SidecarMEMRequest = "50Mi"
)

// GetJenkinsMasterContainerBaseEnvs returns Jenkins master pod envs required by operator
func GetJenkinsMasterContainerBaseEnvs(jenkins *v1alpha2.Jenkins) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
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

	spec := jenkins.Status.Spec
	if spec.ConfigurationAsCode != nil {
		if len(spec.ConfigurationAsCode.Secret.Name) > 0 {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "SECRETS",
				Value: ConfigurationAsCodeSecretVolumePath,
			})
		}

		if spec.ConfigurationAsCode.Enabled {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "CASC_JENKINS_CONFIG",
				Value: ConfigurationAsCodeVolumePath,
			})
		}
	}
	return envVars
}

// getJenkinsHomePath fetches the Home Path for Jenkins
func getJenkinsHomePath(jenkins *v1alpha2.Jenkins) string {
	defaultJenkinsHomePath := "/var/lib/jenkins"
	for _, envVar := range jenkins.Status.Spec.Master.Containers[0].Env {
		if envVar.Name == "JENKINS_HOME" {
			return envVar.Value
		}
	}
	return defaultJenkinsHomePath
}

// GetJenkinsMasterPodBaseVolumes returns Jenkins master pod volumes required by operator
func GetJenkinsMasterPodBaseVolumes(jenkins *v1alpha2.Jenkins) []corev1.Volume {
	volumes := []corev1.Volume{
		getEmptyDirVolume(JenkinsHomeVolumeName),
		getConfigMapVolume(jenkinsScriptsVolumeName, getScriptsConfigMapName(jenkins), 0777),
		getConfigMapVolume(jenkinsInitConfigurationVolumeName, GetInitConfigurationConfigMapName(jenkins)),
		getConfigMapVolume(basePluginsVolumeName, GetBasePluginsVolumeNameConfigMapName(jenkins)),
		getSecretVolume(jenkinsOperatorCredentialsVolumeName, GetOperatorCredentialsSecretName(jenkins)),
	}

	if jenkins.Status != nil && jenkins.Status.Spec != nil && jenkins.Status.Spec.ConfigurationAsCode != nil {
		spec := jenkins.Status.Spec
		configurationAsCode := spec.ConfigurationAsCode
		if configurationAsCode.Enabled {
			// target volume for the init container
			// All casc configmaps will be copied here
			volumes = append(volumes, getEmptyDirVolume(ConfigurationAsCodeVolumeName))
			// volume for default config configmap
			if configurationAsCode.DefaultConfig {
				volumes = append(volumes, getConfigMapVolume(fmt.Sprintf("casc-default-%s", JenkinsDefaultConfigMapName), JenkinsDefaultConfigMapName))
			}
			// Loop to add all casc configmap volumes
			for _, cm := range configurationAsCode.Configurations {
				volumes = append(volumes, getConfigMapVolume(fmt.Sprintf("casc-init-%s", cm.Name), cm.Name))
			}
			// Add casc secret volume
			if len(configurationAsCode.Secret.Name) > 0 {
				volumes = append(volumes, getSecretVolume(ConfigurationAsCodeSecretVolumeName, configurationAsCode.Secret.Name))
			}
		}
	}
	return volumes
}

// GetJenkinsMasterContainerBaseVolumeMounts returns Jenkins master pod volume mounts required by operator
func GetJenkinsMasterContainerBaseVolumeMounts(jenkins *v1alpha2.Jenkins, spec *v1alpha2.JenkinsSpec) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		getVolumeMount(JenkinsHomeVolumeName, getJenkinsHomePath(jenkins), false),
		getVolumeMount(jenkinsScriptsVolumeName, JenkinsScriptsVolumePath, true),
		getVolumeMount(jenkinsInitConfigurationVolumeName, jenkinsInitConfigurationVolumePath, true),
		getSubPathVolumeMount(basePluginsVolumeName, BasePluginsVolumePath, basePluginsFileName, false),
	}

	if spec.ConfigurationAsCode != nil {
		if spec.ConfigurationAsCode.Enabled {
			volumeMounts = append(volumeMounts, getVolumeMount(ConfigurationAsCodeVolumeName, ConfigurationAsCodeVolumePath, false))
			if len(spec.ConfigurationAsCode.Secret.Name) > 0 {
				volumeMounts = append(volumeMounts, getVolumeMount(ConfigurationAsCodeSecretVolumeName, ConfigurationAsCodeSecretVolumePath, false))
			}
		}
	}
	return volumeMounts
}

// NewJenkinsMasterContainer returns Jenkins master Kubernetes container
func NewJenkinsMasterContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	jenkinsContainer := jenkins.Status.Spec.Master.Containers[0]

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

	return GetJenkinsContainer(jenkins, jenkinsContainer, envs)
}

func GetJenkinsContainer(jenkins *v1alpha2.Jenkins, jenkinsContainer v1alpha2.Container, envs []corev1.EnvVar) corev1.Container {
	lifecycle := &corev1.Lifecycle{}
	logger.Info(fmt.Sprintf("ForceBasePluginsInstall value: %+v", jenkins.Spec.ForceBasePluginsInstall))
	if jenkins.Spec.ForceBasePluginsInstall {
		postStartCommand := []string{"bash", "-c", fmt.Sprintf("%s/%s", JenkinsScriptsVolumePath, InitScriptName)}
		logger.Info(fmt.Sprintf("ForceBasePluginsInstall found: Setting up postStart action: %s ", postStartCommand))
		lifecycle = &corev1.Lifecycle{
			PostStart: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: postStartCommand,
				},
			},
		}
	}
	container := corev1.Container{
		Name:            JenkinsMasterContainerName,
		Image:           jenkinsContainer.Image,
		Lifecycle:       lifecycle,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		LivenessProbe:   jenkinsContainer.LivenessProbe,
		ReadinessProbe:  jenkinsContainer.ReadinessProbe,
		Ports: []corev1.ContainerPort{
			GetTCPContainerPort(httpPortName, constants.DefaultHTTPPortInt32),
			GetTCPContainerPort(jnlpPortName, constants.DefaultJNLPPortInt32),
		},
		SecurityContext: jenkinsContainer.SecurityContext,
		Env:             envs,
		Resources:       jenkinsContainer.Resources,
		VolumeMounts:    append(GetJenkinsMasterContainerBaseVolumeMounts(jenkins, jenkins.Status.Spec), jenkinsContainer.VolumeMounts...),
	}
	if jenkinsContainer.Command != nil {
		container.Command = jenkinsContainer.Command
	}

	return container
}

func GetTCPContainerPort(portName string, portNumber int32) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          portName,
		ContainerPort: portNumber,
		Protocol:      corev1.ProtocolTCP,
	}
}

// NewJenkinsConfigContainer returns Jenkins side container for config reloading
func NewJenkinsConfigContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
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
		getVolumeMount(ConfigurationAsCodeVolumeName, ConfigurationAsCodeSecretVolumePath, false),
		getVolumeMount(JenkinsHomeVolumeName, getJenkinsHomePath(jenkins), true),
		getVolumeMount(jenkinsScriptsVolumeName, JenkinsScriptsVolumePath, true),
	}

	return corev1.Container{
		Name:            ConfigSidecarName,
		Image:           getJenkinsSideCarImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars,
		Resources:       GetResourceRequirements(SidecarCPURequest, SidecarMEMRequest, SidecarCPULimit, SidecarMEMLimit),
		VolumeMounts:    volumeMounts,
	}
}

func GetResourceRequirements(cpuRequest string, memRequest string, cpuLimit string, memLimit string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: GetResourceList(cpuRequest, memRequest),
		Limits:   GetResourceList(cpuLimit, memLimit),
	}
}

func GetResourceList(cpu string, mem string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(cpu),
		corev1.ResourceMemory: resource.MustParse(mem),
	}
}

func NewJenkinsBackupContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	backupContainer := corev1.Container{
		Name:            BackupSidecarName,
		Image:           getJenkinsBackupImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", "--"},
		Args:            []string{"while true; do sleep 30; done;"},
		VolumeMounts: []corev1.VolumeMount{
			getVolumeMount(JenkinsBackupVolumeMountName, JenkinsBackupVolumePath, false),
			getVolumeMount(ScriptsVolumeMountName, ScriptsVolumePath, false),
			getVolumeMount(JenkinsHomeVolumeName, getJenkinsHomePath(jenkins), false),
		},
		Stdin: true,
		TTY:   true,
	}

	return backupContainer
}

// NewJenkinsPluginsInitContainer returns Jenkins init container to install required based plugins
func NewJenkinsPluginsInitContainer(spec *v1alpha2.JenkinsSpec) corev1.Container {
	jenkinsContainer := spec.Master.Containers[0]
	volumeMounts := []corev1.VolumeMount{
		getVolumeMount(jenkinsScriptsVolumeName, JenkinsScriptsVolumePath, false),
		getVolumeMount(basePluginsVolumeName, BasePluginsVolumePath, false),
	}
	command := []string{"bash", "-c", "ls"}
	// FIXME use init container to install plugins
	// command := []string{"bash", "-c", fmt.Sprintf("%s/%s", JenkinsScriptsVolumePath, InitScriptName)}
	return corev1.Container{
		Name:            PluginsInitContainerName,
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         command,
		Resources:       GetResourceRequirements(SidecarCPURequest, SidecarMEMRequest, SidecarCPULimit, SidecarMEMLimit),
		VolumeMounts:    volumeMounts,
	}
}

// NewJenkinsConfigInitContainer returns Jenkins init container to copy configmap to make it writable
func NewJenkinsConfigInitContainer(spec *v1alpha2.JenkinsSpec) corev1.Container {
	jenkinsContainer := spec.Master.Containers[0]
	volumeMounts := []corev1.VolumeMount{
		getVolumeMount(ConfigurationAsCodeVolumeName, ConfigurationAsCodeVolumePath, false),
	}

	if spec.ConfigurationAsCode == nil || spec.ConfigurationAsCode.DefaultConfig {
		configurationAsCodeVolumeName := fmt.Sprintf("casc-default-%s", JenkinsDefaultConfigMapName)
		configurationAsCodeVolumePath := jenkinsPath + fmt.Sprintf("/casc-default-%s", JenkinsDefaultConfigMapName)
		volumeMounts = append(volumeMounts, getVolumeMount(configurationAsCodeVolumeName, configurationAsCodeVolumePath, false))
	}

	if spec.ConfigurationAsCode != nil && spec.ConfigurationAsCode.Enabled {
		for _, cm := range spec.ConfigurationAsCode.Configurations {
			configurationAsCodeInitVolumeName := fmt.Sprintf("casc-init-%s", cm.Name)
			configurationAsCodeInitVolumePath := jenkinsPath + fmt.Sprintf("/casc-init-%s", cm.Name)
			volumeMounts = append(volumeMounts, getVolumeMount(configurationAsCodeInitVolumeName, configurationAsCodeInitVolumePath, false))
		}
	}

	command := []string{
		"bash",
		"-c",
		fmt.Sprintf("if [ `ls %s/casc-* > /dev/null 2>&1; echo $?` -eq 0 ]; then find %s/casc-* -type f -exec cp -fL {} %s \\;; fi",
			jenkinsPath, jenkinsPath, ConfigurationAsCodeVolumePath),
	}
	return corev1.Container{
		Name:            ConfigInitContainerName,
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         command,
		Resources:       GetResourceRequirements(SidecarCPURequest, SidecarMEMRequest, SidecarCPULimit, SidecarMEMLimit),
		VolumeMounts:    volumeMounts,
	}
}

// NewJenkinsConfigInitContainer returns Jenkins init container to copy configmap to make it writable
func NewJenkinsBackupInitContainer(spec *v1alpha2.JenkinsSpec) corev1.Container {
	jenkinsContainer := spec.Master.Containers[0]
	volumeMounts := []corev1.VolumeMount{
		getVolumeMount(JenkinsBackupVolumeMountName, JenkinsBackupVolumePath, false),
		getVolumeMount(ScriptsVolumeMountName, ScriptsVolumePath, false),
	}

	scriptTemplate := `cat > %s << %s
SERVER=http://localhost:8080
CRUMB=\$(curl --user \$USER:\$APITOKEN \$SERVER/crumbIssuer/api/xml?xpath=concat\(//crumbRequestField,%%22:%%22,//crumb\)) 
curl -X POST --user \$USER:\$APITOKEN -H "\$CRUMB" \$SERVER/%s
%s
`
	heredoc := "heredoc"
	quietDown := "quietDown"
	cancelQuietDown := "cancelQuietDown"
	restart := "restart"
	safeRestart := "safeRestart"

	quietDownHereDoc := quietDown + heredoc
	cancelQuietDownHereDoc := cancelQuietDown + heredoc
	restartHereDoc := restart + heredoc
	safeRestartHereDoc := safeRestart + heredoc

	quietDownScript := fmt.Sprintf(scriptTemplate, QuietDownScriptPath, quietDownHereDoc, quietDown, quietDownHereDoc)
	cancelQuietDownScript := fmt.Sprintf(scriptTemplate, CancelQuietDownScriptPath, cancelQuietDownHereDoc, cancelQuietDown, cancelQuietDownHereDoc)
	restartScript := fmt.Sprintf(scriptTemplate, RestartScriptPath, restartHereDoc, restart, restartHereDoc)
	safeRestartScript := fmt.Sprintf(scriptTemplate, SafeRestartScriptPath, safeRestartHereDoc, safeRestart, safeRestartHereDoc)

	commandString := fmt.Sprintf(`%s
%s 
%s
%s
`, quietDownScript, cancelQuietDownScript, restartScript, safeRestartScript)

	command := []string{"bash", "-c", commandString}

	return corev1.Container{
		Name:            BackupInitContainerName,
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         command,
		Resources:       GetResourceRequirements(SidecarCPURequest, SidecarMEMRequest, SidecarCPULimit, SidecarMEMLimit),
		VolumeMounts:    volumeMounts,
	}
}

// convertJenkinsContainerToKubernetesContainer converts Jenkins container to Kubernetes container
func convertJenkinsContainerToKubernetesContainer(container v1alpha2.Container) corev1.Container {
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

func newContainers(jenkins *v1alpha2.Jenkins, spec *v1alpha2.JenkinsSpec) (containers []corev1.Container) {
	containers = append(containers, NewJenkinsMasterContainer(jenkins))
	if spec.ConfigurationAsCode != nil {
		if spec.ConfigurationAsCode.Enabled && spec.ConfigurationAsCode.EnableAutoReload {
			containers = append(containers, NewJenkinsConfigContainer(jenkins))
		}
		for _, container := range spec.Master.Containers[1:] {
			containers = append(containers, convertJenkinsContainerToKubernetesContainer(container))
		}
	}
	if spec.BackupEnabled {
		containers = append(containers, NewJenkinsBackupContainer(jenkins))
	}

	return containers
}

func newInitContainers(jenkinsSpec *v1alpha2.JenkinsSpec) (containers []corev1.Container) {
	containers = append(containers, NewJenkinsPluginsInitContainer(jenkinsSpec))
	if jenkinsSpec.ConfigurationAsCode == nil || jenkinsSpec.ConfigurationAsCode.Enabled {
		containers = append(containers, NewJenkinsConfigInitContainer(jenkinsSpec))
	}
	if jenkinsSpec.BackupEnabled {
		containers = append(containers, NewJenkinsBackupInitContainer(jenkinsSpec))
	}
	return containers
}

func GetJenkinsBackupPVCName(jenkins *v1alpha2.Jenkins) string {
	return jenkins.Name + "-jenkins-backup"
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
		case corev1.PodFailed, corev1.PodSucceeded, corev1.PodPending, corev1.PodUnknown:
			return false, conditions.ErrPodCompleted
		}
		return false, nil
	}
}

// return a condition function that indicates whether the given pod is
// currently running
func isPodCompleted(k8sClient client.Client, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod := &corev1.Pod{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: namespace}, pod)
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed, corev1.PodRunning, corev1.PodPending, corev1.PodUnknown:
			return false, nil
		}
		return false, nil
	}
}

// Poll up to timeout seconds for pod to enter running state.
// Returns an error if the pod never enters the running state.
func WaitForPodRunning(k8sClient client.Client, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isPodRunning(k8sClient, podName, namespace))
}

// Poll up to timeout seconds for pod to enter running state.
// Returns an error if the pod never enters the running state.
func WaitForPodIsCompleted(k8sClient client.Client, podName, namespace string) error {
	return wait.PollUntil(time.Second, isPodCompleted(k8sClient, podName, namespace), nil)
}

func getEmptyDirVolume(volumeName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func getConfigMapVolume(volumeName string, configMapName string, mode ...int32) corev1.Volume {
	configMapVolumeSourceDefaultMode := corev1.ConfigMapVolumeSourceDefaultMode
	if len(mode) != 0 {
		configMapVolumeSourceDefaultMode = mode[0]
	}
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: &configMapVolumeSourceDefaultMode,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

func getSecretVolume(volumeName string, secretName string) corev1.Volume {
	secretVolumeSourceDefaultMode := corev1.SecretVolumeSourceDefaultMode
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &secretVolumeSourceDefaultMode,
				SecretName:  secretName,
			},
		},
	}
}

func getSubPathVolumeMount(volumeName string, mountPath string, subPath string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   subPath,
		ReadOnly:  readOnly,
	}
}

func getVolumeMount(volumeName string, mountPath string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
}

// getJenkinsSideCarImage returns the jenkins sidecar image the operator should be using
func getJenkinsSideCarImage() string {
	jenkinsSideCarImage, _ := os.LookupEnv(JenkinsSideCarImageEnvVar)
	if len(jenkinsSideCarImage) == 0 {
		jenkinsSideCarImage = constants.DefaultJenkinsSideCarImage
	}
	return jenkinsSideCarImage
}

// getJenkinsBackupImage returns the ubi minimal image
func getJenkinsBackupImage() string {
	jenkinsBackupImage, _ := os.LookupEnv(JenkinsBackupImageEnvVar)
	if len(jenkinsBackupImage) == 0 {
		jenkinsBackupImage = constants.DefaultJenkinsBackupImage
	}
	return jenkinsBackupImage
}
