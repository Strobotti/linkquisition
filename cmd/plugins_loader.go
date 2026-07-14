//go:build !windows

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/strobotti/linkquisition"
)

// initPluginSupport registers plugin-related CLI commands and flags.
// On non-Windows platforms, plugins are fully supported.
func initPluginSupport() {
	rootCmd.Flags().StringArrayVar(&pluginOpts, "plugin-opt", nil,
		`override plugin settings at runtime (format: plugin.key=value, e.g. shenanigans.effect=matrix)`)
	rootCmd.Flags().BoolVar(&noPlugins, "no-plugins", false,
		`disable all plugin loading (for debugging)`)

	initPluginCmd()
	rootCmd.AddCommand(pluginCmd)
}

// parsePluginOpts parses --plugin-opt flag values ("plugin.key=value") into
// a nested map: map[pluginName]map[settingKey]value.
// Entries that don't match the expected format are silently ignored.
func parsePluginOpts(opts []string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, opt := range opts {
		// Split on first "=" to get "plugin.key" and "value"
		eqIdx := strings.IndexByte(opt, '=')
		if eqIdx < 0 {
			continue
		}
		path := opt[:eqIdx]
		value := opt[eqIdx+1:]

		// Split path on first "." to get plugin name and key
		dotIdx := strings.IndexByte(path, '.')
		if dotIdx < 0 || dotIdx == 0 || dotIdx == len(path)-1 {
			continue
		}
		pluginName := path[:dotIdx]
		key := path[dotIdx+1:]

		if result[pluginName] == nil {
			result[pluginName] = make(map[string]string)
		}
		result[pluginName][key] = value
	}
	return result
}

func setupPlugins(
	settingsService linkquisition.SettingsService,
	pluginServiceProvider linkquisition.PluginServiceProvider,
	logger *slog.Logger,
	overrides map[string]map[string]string,
) []linkquisition.Plugin {
	settings := settingsService.GetSettings()
	var plugins []linkquisition.Plugin

	for _, pluginSettings := range settings.Plugins {
		if pluginSettings.IsDisabled {
			logger.Debug("Plugin is disabled by configuration directive", "plugin", pluginSettings.Path)
			continue
		}

		pluginPath := pluginSettings.Path
		if !strings.HasSuffix(pluginPath, pluginExtension) {
			pluginPath += pluginExtension
		}

		if _, err := os.Stat(pluginPath); err != nil {
			pluginPathToCheck := filepath.Join(settingsService.GetPluginFolderPath(), pluginPath)
			if _, err := os.Stat(pluginPathToCheck); err == nil {
				pluginPath = pluginPathToCheck
			} else {
				logger.Error("Error loading plugin: file not found", "plugin", pluginSettings.Path, "checked", pluginPathToCheck, "error", err.Error())
				continue
			}
		}

		plug, err := openPlugin(pluginPath, logger)
		if plug == nil || err != nil {
			logger.Error("Error opening plugin", "plugin", pluginSettings.Path, "path", pluginPath, "error", err.Error())
			continue
		}

		// Apply runtime overrides from --plugin-opt flags
		effectiveSettings := pluginSettings.Settings
		pluginName := strings.TrimSuffix(filepath.Base(pluginSettings.Path), pluginExtension)
		if opts, ok := overrides[pluginName]; ok {
			if effectiveSettings == nil {
				effectiveSettings = make(map[string]interface{})
			}
			for k, v := range opts {
				effectiveSettings[k] = v
				logger.Debug("Plugin setting overridden via --plugin-opt", "plugin", pluginName, "key", k, "value", v)
			}
		}

		if p, err := setupPlugin(plug, effectiveSettings, pluginServiceProvider); err != nil {
			logger.Error("Error setting up plugin", "plugin", pluginSettings.Path, "error", err.Error())
		} else {
			logger.Debug("Plugin loaded successfully", "plugin", pluginSettings.Path)
			plugins = append(plugins, p)
		}
	}

	return plugins
}

// openPlugin wraps plugin.Open with panic recovery — Go plugin loading can panic
// on interface mismatches or incompatible builds rather than returning an error.
func openPlugin(path string, logger *slog.Logger) (p *plugin.Plugin, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while opening plugin %s: %v", path, r)
			logger.Error("Plugin open panicked", "path", path, "panic", fmt.Sprintf("%v", r))
		}
	}()

	p, err = plugin.Open(path)
	return p, err
}

func setupPlugin(
	plug *plugin.Plugin,
	settings map[string]interface{},
	pluginServiceProvider linkquisition.PluginServiceProvider,
) (
	p linkquisition.Plugin,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while setting up plugin: %v", r)
		}
	}()

	var symbol plugin.Symbol
	symbol, err = plug.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin symbol lookup returned an error: %v", err)
	}

	var ok bool
	p, ok = symbol.(linkquisition.Plugin)
	if !ok {
		return nil, fmt.Errorf(
			"plugin symbol does not implement linkquisition.Plugin interface (got %T) — "+
				"this usually means the plugin was built against a different version of the app",
			symbol,
		)
	}

	if setupErr := p.Setup(pluginServiceProvider, settings); setupErr != nil {
		return nil, fmt.Errorf("plugin setup failed: %w", setupErr)
	}

	return p, nil
}

// probePluginMetadata opens a plugin .so file and retrieves its Metadata without calling Setup.
// Used by the configurator to display plugin info without fully initializing plugins.
func probePluginMetadata(path string, logger *slog.Logger) (meta linkquisition.PluginMetadata, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while probing plugin %s: %v", path, r)
		}
	}()

	plug, openErr := openPlugin(path, logger)
	if plug == nil || openErr != nil {
		return meta, fmt.Errorf("failed to open plugin: %w", openErr)
	}

	symbol, lookupErr := plug.Lookup("Plugin")
	if lookupErr != nil {
		return meta, fmt.Errorf("plugin symbol lookup failed: %w", lookupErr)
	}

	p, ok := symbol.(linkquisition.Plugin)
	if !ok {
		return meta, fmt.Errorf("plugin does not implement Plugin interface (got %T)", symbol)
	}

	return p.Metadata(), nil
}
