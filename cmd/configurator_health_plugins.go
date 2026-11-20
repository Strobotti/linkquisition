//go:build !windows

package main

import (
	"errors"
	"fmt"

	"github.com/strobotti/linkquisition"
)

// probePluginHealth opens the configured plugin without calling Setup. Health checks
// receive the saved settings rather than the possibly stale running plugin state.
func (c *Configurator) probePluginHealth(
	settings linkquisition.PluginSettings, configDir string,
) (issues []linkquisition.PluginHealthIssue, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("plugin health check panicked: %v", recovered)
		}
	}()
	path := c.resolvePluginPath(settings.Path)
	if path == "" {
		return nil, errors.New("plugin file not found")
	}
	plug, err := openPlugin(path, c.logger)
	if err != nil {
		return nil, fmt.Errorf("open plugin: %w", err)
	}
	symbol, err := plug.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("look up plugin: %w", err)
	}
	return checkPluginHealth(symbol, settings.Settings, configDir)
}

// checkPluginHealth dispatches optional checks without calling the plugin's Setup method.
func checkPluginHealth(symbol interface{}, settings map[string]interface{}, configDir string) ([]linkquisition.PluginHealthIssue, error) {
	checker, ok := symbol.(linkquisition.PluginHealthChecker)
	if !ok {
		return nil, nil
	}
	return checker.HealthCheck(settings, configDir)
}
