package resources

import (
	"fmt"
	"os"

	jenkinsv1alpha2 "github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	NameWithSuffixFormat         = "%s-%s"
	PluginDefinitionFormat       = "%s:%s"
	BuilderDockerfileArg         = "--dockerfile=/dockerfile/Dockerfile"
	BuilderContextDirArg         = "--context=dir://workspace/"
	BuilderNoPushArg             = "--no-push"
	BuilderDestinationArg        = "--destination"
	BuilderDigestFileArg         = "--digest-file=/dev/termination-log"
	BuilderReproducibleArg       = "--reproducible"
	BuilderSuffix                = "builder"
	DockerfileStorageSuffix      = "dockerfile-storage"
	DockerSecretSuffix           = "docker-secret"
	DockerfileNameSuffix         = "dockerfile"
	JenkinsImageBuilderImage     = "quay.io/redhat-developer/openshift-jenkins-image-builder:v1.3.0"
	JenkinsImageBuilderName      = "jenkins-image-builder"
	JenkinsImageDefaultBaseImage = "jenkins/jenkins:lts"
	DefaultImageRegistry         = "image-registry.openshift-image-registry.svc:5000"
	DefaultImageTag              = "latest"
	DockerfileName               = "Dockerfile"
	DockerfileTemplate           = `FROM %s
RUN curl -o /tmp/install-plugins.sh https://raw.githubusercontent.com/jenkinsci/docker/master/install-plugins.sh
RUN chmod +x /tmp/install-plugins.sh
RUN install-plugins.sh %s `
)

var log = logf.Log.WithName("controller_jenkinsimage")

// NewBuilderPod returns a busybox pod with the same name/namespace as the cr.
func NewBuilderPod(client client.Client, cr *jenkinsv1alpha2.JenkinsImage) *corev1.Pod {
	logger := log.WithName("jenkinsimage_NewBuilderPod")
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, BuilderSuffix)
	logger.Info(fmt.Sprintf("Creating a new builder pod with name %s", name))
	builderPodArgs := []string{BuilderReproducibleArg, BuilderDockerfileArg, BuilderContextDirArg, BuilderDigestFileArg}
	//--no-push, or the destination specified in the CR, or the openshift default registry if we are on OpenShift
	destinationArg := GetDestinationRepository(cr)
	builderPodArgs = append(builderPodArgs, destinationArg)
	volumes := getVolumes(cr, client)
	volumeMounts := getVolumesMounts(cr, client)
	p := getBuilderPod(name, cr, builderPodArgs, volumeMounts, volumes)
	return p
}

func getBuilderPod(name string, cr *jenkinsv1alpha2.JenkinsImage, builderPodArgs []string, volumeMounts []corev1.VolumeMount, volumes []corev1.Volume) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:         JenkinsImageBuilderName,
					Image:        getJenkinsImageBuilderImage(),
					Args:         builderPodArgs,
					VolumeMounts: volumeMounts,
				},
			},
			Volumes: volumes,
		},
	}
}

const JenkinsImageBuilderEnv = "JENKINS_IMAGE_BUILDER_NAME"

func getJenkinsImageBuilderImage() string {
	image := os.Getenv(JenkinsImageBuilderEnv)
	if len(image) != 0 {
		return image
	}
	return JenkinsImageBuilderImage
}

// GetDestinationRepository returns the destination repository matching To.Repository/To.Name:To.Tag or default to
// openshift-image-registry if running on openshift or --no-push otherwise
func GetDestinationRepository(cr *jenkinsv1alpha2.JenkinsImage) string {
	logger := log.WithName("jenkinsimage_GetDestinationRepository")
	spec := cr.Spec
	destination := spec.To
	repositoryArg := ""
	if len(destination.Registry) == 0 {
		if IsImageRegistryAvailableCached() { // on OpenShift get the default registry
			repositoryArg = fmt.Sprintf("%s=%s/%s/%s:%s", BuilderDestinationArg, DefaultImageRegistry, cr.Namespace, cr.Name, DefaultImageTag)
		} else {
			repositoryArg = BuilderNoPushArg
			logger.Info(fmt.Sprintf("No Spec.Destnation.Registry found in JenkinsImage %s: push will not be performed", cr.Name))
		}
	} else {
		repositoryArg = fmt.Sprintf("%s=%s/%s:%s", BuilderDestinationArg, destination.Registry, destination.Name, destination.Tag)
		logger.Info(fmt.Sprintf("Builder pod will push to %s using secret %s", repositoryArg, spec.To.Secret))
	}

	return repositoryArg
}

// NewDockerfileConfigMap returns a busybox pod with the same name/namespace as the cr.
func NewDockerfileConfigMap(cr *jenkinsv1alpha2.JenkinsImage) *corev1.ConfigMap {
	dockerfileContent := fmt.Sprintf(DockerfileTemplate, getDefaultedBaseImage(cr), getJenkinsPluginsList(cr))
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	data := map[string]string{DockerfileName: dockerfileContent}
	dockerfile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Data: data,
	}

	return dockerfile
}

func getJenkinsPluginsList(cr *jenkinsv1alpha2.JenkinsImage) string {
	logger := log.WithName("jenkinsimage_getPluginsList")
	plugins := ""
	for _, v := range cr.Spec.Plugins {
		plugins += fmt.Sprintf(PluginDefinitionFormat, v.Name, v.Version) + " "
		logger.Info(fmt.Sprintf("Adding plugin %s:%s ", v.Name, v.Version))
	}

	return plugins
}

func getDefaultedBaseImage(cr *jenkinsv1alpha2.JenkinsImage) string {
	if len(cr.Spec.From.Name) != 0 {
		return cr.Spec.From.Name
	}

	return JenkinsImageDefaultBaseImage
}

func getVolumes(instance *jenkinsv1alpha2.JenkinsImage, client client.Client) []corev1.Volume {
	logger := log.WithName("jenkinsimage_getVolumes")
	volumes := []corev1.Volume{}
	logger.Info(fmt.Sprintf("Creating volumes for  instance:  %s ", instance.Name))
	name := fmt.Sprintf(NameWithSuffixFormat, instance.Name, DockerfileStorageSuffix)
	storage := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	volumes = append(volumes, storage)
	logger.Info(fmt.Sprintf("Storage volume of type emptyDir and name :  %s created", name))

	configMapVolumeName := GetDockerfileConfigMapName(instance)
	config := corev1.Volume{
		Name: configMapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapVolumeName},
			},
		},
	}
	logger.Info(fmt.Sprintf("RestConfig volume of type ConfigMap and name :  %s created", name))
	volumes = append(volumes, config)

	name = fmt.Sprintf(NameWithSuffixFormat, instance.Name, DockerSecretSuffix)
	secretName, err := getPushSecretName(instance, client)
	logger.Info(fmt.Sprintf("Push to registry will be using secret:  %s ", secretName))
	if len(secretName) == 0 || err != nil {
		logger.Info("No secret found: Secret Volume will not be added")
	} else {
		secret := corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}
		logger.Info(fmt.Sprintf("Secret volume of type Secret using secret %s and name : %s created", instance.Spec.To.Secret, name))
		volumes = append(volumes, secret)
	}
	return volumes
}

func GetDockerfileConfigMapName(cr *jenkinsv1alpha2.JenkinsImage) string {
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	return name
}

func getPushSecretName(cr *jenkinsv1alpha2.JenkinsImage, clientSet client.Client) (string, error) {
	if len(cr.Spec.To.Secret) != 0 {
		return cr.Spec.To.Secret, nil
	}
	return GetDockerBuilderSecretName(cr.Namespace, clientSet)
}

func getVolumesMounts(cr *jenkinsv1alpha2.JenkinsImage, client client.Client) []corev1.VolumeMount {
	logger := log.WithName("jenkinsimage_getVolumesMounts")
	volumeMounts := []corev1.VolumeMount{}

	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileStorageSuffix)
	mountPath := "/workspace"
	storage := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
	volumeMounts = append(volumeMounts, storage)
	logger.Info(fmt.Sprintf("Volument mount for %s and mountPath : %s created", name, mountPath))

	name = fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	mountPath = "/dockerfile"
	config := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
	volumeMounts = append(volumeMounts, config)
	logger.Info(fmt.Sprintf("Volument mount for %s and mountPath : %s created", name, mountPath))

	pushSecretName, err := getPushSecretName(cr, client)
	if len(pushSecretName) != 0 && err == nil {
		name = fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerSecretSuffix)
		mountPath = "/kaniko/.docker/"
		secret := corev1.VolumeMount{
			Name:      name,
			MountPath: mountPath,
		}
		volumeMounts = append(volumeMounts, secret)
		logger.Info(fmt.Sprintf("Volument mount for %s and mountPath : %s created", name, mountPath))
	}
	return volumeMounts
}
