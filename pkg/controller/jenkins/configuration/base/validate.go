package base

import (
	"fmt"
	"regexp"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
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

	if !r.validatePlugins(jenkins.Spec.Master.Plugins) {
		return false, nil
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(pluginsWithVersions map[string][]string) bool {
	valid := true
	allPlugins := map[string][]plugins.Plugin{}

	for rootPluginName, dependentPluginNames := range pluginsWithVersions {
		if _, err := plugins.New(rootPluginName); err != nil {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid root plugin name '%s'", rootPluginName))
			valid = false
		}

		dependentPlugins := []plugins.Plugin{}
		for _, pluginName := range dependentPluginNames {
			if p, err := plugins.New(pluginName); err != nil {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Invalid dependent plugin name '%s' in root plugin '%s'", pluginName, rootPluginName))
				valid = false
			} else {
				dependentPlugins = append(dependentPlugins, *p)
			}
		}

		allPlugins[rootPluginName] = dependentPlugins
	}

	if valid {
		return plugins.VerifyDependencies(allPlugins)
	}

	return valid
}
