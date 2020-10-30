package base

import (
	"github.com/bndr/gojenkins"
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
)

func isValidPlugin(plugin gojenkins.Plugin) bool {
	return plugin.Active && plugin.Enabled && !plugin.Deleted
}

func isPluginInstalled(plugins *gojenkins.Plugins, requiredPlugin v1alpha2.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, isValidPlugin(*p)
}
