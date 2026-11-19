//go:build windows

package main

import "github.com/strobotti/linkquisition"

// Windows does not support Go plugins.
func (c *Configurator) probePluginHealth(_ linkquisition.PluginSettings, _ string) ([]linkquisition.PluginHealthIssue, error) {
	return nil, nil
}
