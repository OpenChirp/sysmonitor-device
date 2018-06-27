package plugins

import (
	"fmt"
	"plugin"

	"github.com/sirupsen/logrus"
)

const pluginNewObjectFunction = "NewPlugin"

// Plugin sets the interface that a plugin must provide
type Plugin interface {
	GetReport(log *logrus.Entry) map[string]string
}

// PluginManager holds the context of all loaded plugins
type PluginManager struct {
	pluginNames []string
	plugins     []Plugin
}

// NewPluginManger tries to load all plugin referenced in `paths` and returns
// a context that allows fetching reports from them
func NewPluginManger(paths []string) (*PluginManager, error) {
	pm := new(PluginManager)
	pm.plugins = make([]Plugin, len(paths))
	pm.pluginNames = make([]string, len(paths))

	for i, path := range paths {
		p, err := plugin.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Failed to open plugin %s: %v", path, err)
		}

		sym, err := p.Lookup(pluginNewObjectFunction)
		if err != nil {
			return nil, fmt.Errorf("Failed to find %s function for plugin %s: %v", pluginNewObjectFunction, path, err)
		}
		newplugin, ok := sym.(func() (Plugin, error))
		if !ok {
			return nil, fmt.Errorf("%s function type is invalid for plugin %s: %v", pluginNewObjectFunction, path, err)
		}

		pobject, err := newplugin()
		if err != nil {
			return nil, fmt.Errorf("%s function for plugin %s gave error: %v", pluginNewObjectFunction, path, err)
		}
		pm.plugins[i] = pobject
		pm.pluginNames[i] = path
	}
	return pm, nil
}

// GetReports rounds up all the reports from the loaded plugins
func (pm *PluginManager) GetReports(log *logrus.Logger) map[string]string {
	allvalues := make(map[string]string)
	for i, p := range pm.plugins {
		logitem := log.WithField("plugin", pm.pluginNames[i])
		for k, v := range p.GetReport(logitem) {
			allvalues[k] = v
		}
	}
	return allvalues
}
