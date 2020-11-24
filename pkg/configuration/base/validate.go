package base

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	docker "github.com/docker/distribution/reference"
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)

var defaultConfigMap = &corev1.ConfigMap{
	Data: map[string]string{
		"01-basic-settings.yaml": `
groovy:
- script: >
    import jenkins.model.Jenkins;
    import jenkins.model.JenkinsLocationConfiguration;
    import hudson.model.Node.Mode;

    def jenkins = Jenkins.instance;
    //Number of jobs that run simultaneously on master.
    jenkins.setNumExecutors(6);
    //Jobs must specify that they want to run on master
    jenkins.setMode(Mode.EXCLUSIVE);
    jenkins.save();`,

		"02-enable-csrf.yaml": `
groovy:
- script: >
    import hudson.security.csrf.DefaultCrumbIssuer;
    import jenkins.model.Jenkins;

    def jenkins = Jenkins.instance;

    if (jenkins.getCrumbIssuer() == null) {
      jenkins.setCrumbIssuer(new DefaultCrumbIssuer(true));
      jenkins.save();
      println('CSRF Protection enabled.');
    } else {
      println('CSRF Protection already configured.');
    }`,

		"03-disable-stats.yaml": `
groovy:
- script: >
    import jenkins.model.Jenkins;

    def jenkins = Jenkins.instance;

    if (jenkins.isUsageStatisticsCollected()) {
      jenkins.setNoUsageStatistics(true);
      jenkins.save();
      println('Jenkins usage stats submitting disabled.');
    } else {
      println('Nothing changed.  Usage stats are not submitted to the Jenkins project.');                                                                              
    }`,

		"04-enable-master-control.yaml": `
groovy:
- script: >
    import jenkins.security.s2m.AdminWhitelistRule;
    import jenkins.model.Jenkins;

    // see https://wiki.jenkins-ci.org/display/JENKINS/Slave+To+Master+Access+Control                                                                                    
    def jenkins = Jenkins.instance;
    jenkins.getInjector().getInstance(AdminWhitelistRule.class).setMasterKillSwitch(false); // for real though, false equals enabled..........                           
    jenkins.save();`,

		"05-disable-insecure.yaml": `
groovy:
- script: >
    import jenkins.*;
    import jenkins.model.*;
    import hudson.model.*;
    import jenkins.security.s2m.*;

    def jenkins = Jenkins.instance;

    println("Disabling insecure Jenkins features...");

    println("Disabling insecure protocols...");
    println("Old protocols: [" + jenkins.getAgentProtocols().join(", ") + "]");
    HashSet<String> newProtocols = new HashSet<>(jenkins.getAgentProtocols());
    newProtocols.removeAll(Arrays.asList("JNLP3-connect", "JNLP2-connect", "JNLP-connect", "CLI-connect"));                                                              
    println("New protocols: [" + newProtocols.join(", ") + "]");
    jenkins.setAgentProtocols(newProtocols);

    println("Disabling CLI access of /cli URL...");
    def remove = { list ->
      list.each { item ->
        if (item.getClass().name.contains("CLIAction")) {
          println("Removing extension ${item.getClass().name}")
          list.remove(item)
        }
      }
    };
    remove(jenkins.getExtensionList(RootAction.class));
    remove(jenkins.actions);

    if (jenkins.getDescriptor("jenkins.CLI") != null) {
      jenkins.getDescriptor("jenkins.CLI").get().setEnabled(false);
    }

    jenkins.save();`,

		"06-configure-views.yaml": `
groovy:
- script: >
    import hudson.model.ListView;
    import jenkins.model.Jenkins;

    def Jenkins jenkins = Jenkins.getInstance();

    def seedViewName = 'seed-jobs';
    def nonSeedViewName = 'non-seed-jobs';

    if (jenkins.getView(seedViewName) == null) {
      def seedView = new ListView(seedViewName);
      seedView.setIncludeRegex('.*job-dsl-seed.*');
      jenkins.addView(seedView);
    }

    if (jenkins.getView(nonSeedViewName) == null) {
      def nonSeedView = new ListView(nonSeedViewName);
      nonSeedView.setIncludeRegex('((?!seed)(?!jenkins).)*');
      jenkins.addView(nonSeedView);
    }

    jenkins.save();`,

		"07-disable-dsl-approval.yaml": `
groovy:
- script: >
    import jenkins.model.Jenkins;
    import javaposse.jobdsl.plugin.GlobalJobDslSecurityConfiguration;
    import jenkins.model.GlobalConfiguration;

    // disable Job DSL script approval
    GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).useScriptSecurity=false;                                                                      
    GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).save();`,
	},
}

// Validate validates Jenkins CR Spec.master section
func (r *JenkinsBaseConfigurationReconciler) Validate(jenkins *v1alpha2.Jenkins) ([]string, error) {
	var messages []string

	if msg := r.validateReservedVolumes(); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg, err := r.validateVolumes(); err != nil {
		return nil, err
	} else if len(msg) > 0 {
		messages = append(messages, msg...)
	}

	actualSpec := jenkins.Status.Spec
	for _, container := range actualSpec.Master.Containers {
		if msg := r.validateContainer(container); len(msg) > 0 {
			for _, m := range msg {
				messages = append(messages, fmt.Sprintf("Container `%s` - %s", container.Name, m))
			}
		}
	}

	if msg := r.validatePlugins(plugins.BasePlugins(), actualSpec.Master.BasePlugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg := r.validateJenkinsMasterPodEnvs(); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg, err := r.validateConfiguration(actualSpec.ConfigurationAsCode, jenkins.Name); err != nil {
		return nil, err
	} else if len(msg) > 0 {
		messages = append(messages, msg...)
	}

	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateJenkinsMasterContainerCommand() []string {
	masterContainer := r.Configuration.GetJenkinsMasterContainer()
	if masterContainer == nil || masterContainer.Command == nil {
		return []string{}
	}

	jenkinsOperatorInitScript := fmt.Sprintf("%s/%s && ", resources.JenkinsScriptsVolumePath, resources.InitScriptName)
	correctCommand := []string{
		"bash",
		"-c",
		fmt.Sprintf("%s<optional-custom-command> && exec <command-which-start-jenkins>", jenkinsOperatorInitScript),
	}
	invalidCommandMessage := []string{fmt.Sprintf("spec.master.containers[%s].command is invalid, make sure it looks like '%v', otherwise the operator won't configure default user and install plugins. 'exec' is required to propagate signals to the Jenkins.", masterContainer.Name, correctCommand)}
	if masterContainer.Command[0] != correctCommand[0] {
		return invalidCommandMessage
	}
	if masterContainer.Command[1] != correctCommand[1] {
		return invalidCommandMessage
	}
	if !strings.HasPrefix(masterContainer.Command[2], jenkinsOperatorInitScript) {
		return invalidCommandMessage
	}
	if !strings.Contains(masterContainer.Command[2], "exec") {
		return invalidCommandMessage
	}

	return []string{}
}

func (r *JenkinsBaseConfigurationReconciler) validateImagePullSecrets() ([]string, error) {
	var messages []string
	actualSpec := r.Configuration.Jenkins.Status.Spec

	for _, sr := range actualSpec.Master.ImagePullSecrets {
		msg, err := r.validateImagePullSecret(sr.Name)
		if err != nil {
			return nil, err
		}
		if len(msg) > 0 {
			messages = append(messages, msg...)
		}
	}
	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateImagePullSecret(secretName string) ([]string, error) {
	var messages []string
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, secret)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("Secret %s not found defined in spec.master.imagePullSecrets", secretName))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	if secret.Data["docker-server"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-server' key.", secretName))
	}
	if secret.Data["docker-username"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-username' key.", secretName))
	}
	if secret.Data["docker-password"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-password' key.", secretName))
	}
	if secret.Data["docker-email"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-email' key.", secretName))
	}
	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateVolumes() ([]string, error) {
	var messages []string
	actualSpec := r.Configuration.Jenkins.Status.Spec
	for _, volume := range actualSpec.Master.Volumes {
		switch {
		case volume.ConfigMap != nil:
			if msg, err := r.validateConfigMapVolume(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		case volume.Secret != nil:
			if msg, err := r.validateSecretVolume(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		case volume.PersistentVolumeClaim != nil:
			if msg, err := r.validatePersistentVolumeClaim(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		}
	}

	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validatePersistentVolumeClaim(volume corev1.Volume) ([]string, error) {
	var messages []string

	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.PersistentVolumeClaim.ClaimName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, pvc)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("PersistentVolumeClaim '%s' not found for volume '%v'", volume.PersistentVolumeClaim.ClaimName, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateConfigMapVolume(volume corev1.Volume) ([]string, error) {
	var messages []string
	if volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional {
		return nil, nil
	}

	configMap := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.ConfigMap.Name, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, configMap)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("ConfigMap '%s' not found for volume '%v'", volume.ConfigMap.Name, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateSecretVolume(volume corev1.Volume) ([]string, error) {
	var messages []string
	if volume.Secret.Optional != nil && *volume.Secret.Optional {
		return nil, nil
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.Secret.SecretName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, secret)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("Secret '%s' not found for volume '%v'", volume.Secret.SecretName, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *JenkinsBaseConfigurationReconciler) validateReservedVolumes() []string {
	var messages []string

	for _, baseVolume := range resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins) {
		actualSpec := r.Configuration.Jenkins.Status.Spec
		for _, volume := range actualSpec.Master.Volumes {
			if baseVolume.Name == volume.Name {
				messages = append(messages, fmt.Sprintf("Jenkins Master pod volume '%s' is reserved please choose different one", volume.Name))
			}
		}
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) validateContainer(container v1alpha2.Container) []string {
	var messages []string
	if container.Image == "" {
		messages = append(messages, "Image not set")
	}

	if !dockerImageRegexp.MatchString(container.Image) && !docker.ReferenceRegexp.MatchString(container.Image) {
		messages = append(messages, "Invalid image")
	}

	if container.ImagePullPolicy == "" {
		messages = append(messages, "Image pull policy not set")
	}

	if msg := r.validateContainerVolumeMounts(container); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) validateContainerVolumeMounts(container v1alpha2.Container) []string {
	var messages []string
	allVolumes := append(resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins), r.Configuration.Jenkins.Status.Spec.Master.Volumes...)

	for _, volumeMount := range container.VolumeMounts {
		if len(volumeMount.MountPath) == 0 {
			messages = append(messages, fmt.Sprintf("mountPath not set for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
		}

		foundVolume := false
		for _, volume := range allVolumes {
			if volumeMount.Name == volume.Name {
				foundVolume = true
			}
		}

		if !foundVolume {
			messages = append(messages, fmt.Sprintf("Not found volume for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
		}
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) validateJenkinsMasterPodEnvs() []string {
	var messages []string
	baseEnvs := resources.GetJenkinsMasterContainerBaseEnvs(r.Configuration.Jenkins)
	baseEnvNames := map[string]string{}
	for _, env := range baseEnvs {
		baseEnvNames[env.Name] = env.Value
	}

	javaOpts := corev1.EnvVar{}
	actualSpec := r.Configuration.Jenkins.Status.Spec
	for _, userEnv := range actualSpec.Master.Containers[0].Env {
		if userEnv.Name == constants.JavaOpsVariableName {
			javaOpts = userEnv
		}
		// if _, overriding := baseEnvNames[userEnv.Name]; overriding {
		//	messages = append(messages, fmt.Sprintf("Jenkins Master container env '%s' cannot be overridden", userEnv.Name))
		//}
	}

	requiredFlags := map[string]bool{
		"-Djenkins.install.runSetupWizard=false": false,
		"-Djava.awt.headless=true":               false,
	}
	for _, setFlag := range strings.Split(javaOpts.Value, " ") {
		for requiredFlag := range requiredFlags {
			if setFlag == requiredFlag {
				requiredFlags[requiredFlag] = true

				break
			}
		}
	}
	for requiredFlag, set := range requiredFlags {
		if !set {
			messages = append(messages, fmt.Sprintf("Jenkins Master container env '%s' doesn't have required flag '%s'", constants.JavaOpsVariableName, requiredFlag))
		}
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) validatePlugins(requiredBasePlugins []plugins.Plugin, basePlugins []v1alpha2.Plugin) []string {
	var messages []string
	allPlugins := map[plugins.Plugin][]plugins.Plugin{}

	for _, jenkinsPlugin := range basePlugins {
		plugin, err := plugins.NewPlugin(jenkinsPlugin.Name, jenkinsPlugin.Version, jenkinsPlugin.DownloadURL)
		if err != nil {
			messages = append(messages, err.Error())
		}

		if plugin != nil {
			allPlugins[*plugin] = []plugins.Plugin{}
		}
	}

	if msg := plugins.VerifyDependencies(allPlugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg := r.verifyBasePlugins(requiredBasePlugins, basePlugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) verifyBasePlugins(requiredBasePlugins []plugins.Plugin, basePlugins []v1alpha2.Plugin) []string {
	var messages []string

	for _, requiredBasePlugin := range requiredBasePlugins {
		found := false
		for _, basePlugin := range basePlugins {
			if requiredBasePlugin.Name == basePlugin.Name {
				found = true

				break
			}
		}
		if !found {
			messages = append(messages, fmt.Sprintf("Missing plugin '%s' in spec.master.basePlugins", requiredBasePlugin.Name))
		}
	}

	return messages
}

func (r *JenkinsBaseConfigurationReconciler) validateConfiguration(configuration *v1alpha2.Configuration, name string) ([]string, error) {
	var messages []string
	if configuration == nil {
		return messages, nil
	}

	if len(configuration.Secret.Name) > 0 && len(configuration.Configurations) == 0 {
		messages = append(messages, fmt.Sprintf("%s.secret.name is set but %s.configurations is empty", name, name))
	}
	jenkinsInstanceNamespace := r.Configuration.Jenkins.ObjectMeta.Namespace
	if len(configuration.Secret.Name) > 0 {
		secret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configuration.Secret.Name, Namespace: jenkinsInstanceNamespace}, secret)
		if err != nil {
			messages = append(messages, fmt.Sprintf("Secret '%s' configured in %s.secret.name but not found", configuration.Secret.Name, name))
			return messages, stackerr.WithStack(err)
		}
	}

	for index, configMapRef := range configuration.Configurations {
		if len(configMapRef.Name) == 0 {
			messages = append(messages, fmt.Sprintf("%s.configurations[%d] name is empty", name, index))
			continue
		}

		configMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: jenkinsInstanceNamespace}, configMap)
		if err != nil {
			messages = append(messages, fmt.Sprintf("ConfigMap '%s' configured in %s.configurations[%d] but not found", configMapRef.Name, name, index))
			return messages, stackerr.WithStack(err)
		}
	}
	if configuration.DefaultConfig {
		configMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: resources.JenkinsDefaultConfigMapName, Namespace: jenkinsInstanceNamespace}, configMap)
		if err != nil {
			if apierrors.IsNotFound(err) {
				r.logger.Info(fmt.Sprintf("Default config is enabled but Default ConfigMap '%s' is not found, creating Default ConfigMap ", resources.JenkinsDefaultConfigMapName))
				defaultConfigMap.Namespace = jenkinsInstanceNamespace
				defaultConfigMap.Name = resources.JenkinsDefaultConfigMapName
				err = r.Client.Create(context.TODO(), defaultConfigMap)
				if err != nil {
					messages = append(messages, fmt.Sprintf("Not able to create Default ConfigMap %s", resources.JenkinsDefaultConfigMapName))
				}
			}
			return messages, stackerr.WithStack(err)
		}
	}
	return messages, nil
}
