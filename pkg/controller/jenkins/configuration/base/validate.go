package base

import (
	"fmt"
	"regexp"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	docker "github.com/docker/distribution/reference"
)

var (
	dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)
)

// Validate validates Jenkins CR Spec.master section
func (r *ReconcileJenkinsBaseConfiguration) Validate(jenkins *v1alpha1.Jenkins) (bool, error) {
	if jenkins.Spec.Master.Image == "" {
		r.logger.V(log.VWarn).Info("Image not set")
		return false, nil
	}

	if !dockerImageRegexp.MatchString(jenkins.Spec.Master.Image) && !docker.ReferenceRegexp.MatchString(jenkins.Spec.Master.Image) {
		r.logger.V(log.VWarn).Info("Invalid image")
		return false, nil

	}

	if !r.validatePlugins(jenkins.Spec.Master.OperatorPlugins, jenkins.Spec.Master.Plugins) {
		return false, nil
	}

	if !r.validateJenkinsMasterPodEnvs() {
		return false, nil
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateJenkinsMasterPodEnvs() bool {
	baseEnvs := resources.GetJenkinsMasterPodBaseEnvs()
	baseEnvNames := map[string]string{}
	for _, env := range baseEnvs {
		baseEnvNames[env.Name] = env.Value
	}

	valid := true
	for _, userEnv := range r.jenkins.Spec.Master.Env {
		if _, overriding := baseEnvNames[userEnv.Name]; overriding {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Jenkins Master pod env '%s' cannot be overridden", userEnv.Name))
			valid = false
		}
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(pluginsWithVersionSlice ...map[string][]string) bool {
	valid := true
	allPlugins := map[plugins.Plugin][]plugins.Plugin{}

	for _, pluginsWithVersions := range pluginsWithVersionSlice {
		for rootPluginName, dependentPluginNames := range pluginsWithVersions {
			rootPlugin, err := plugins.New(rootPluginName)
			if err != nil {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid root plugin name '%s'", rootPluginName))
				valid = false
			}

			var dependentPlugins []plugins.Plugin
			for _, pluginName := range dependentPluginNames {
				if p, err := plugins.New(pluginName); err != nil {
					r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid dependent plugin name '%s' in root plugin '%s'", pluginName, rootPluginName))
					valid = false
				} else {
					dependentPlugins = append(dependentPlugins, *p)
				}
			}

			if rootPlugin != nil {
				allPlugins[*rootPlugin] = dependentPlugins
			}
		}
	}

	if valid {
		return plugins.VerifyDependencies(allPlugins)
	}

	return valid
}
