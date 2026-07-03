package linkquisition

import (
	"context"
	"log/slog"
)

// PluginServiceProvider is an interface that provides the logger and settings to the plugin
// This is passed to the plugin as a dependency when being setup.
type PluginServiceProvider interface {
	GetLogger() *slog.Logger
	GetSettings() *Settings
	GetConfigFolderPath() string
}

// Plugin is an interface that all plugins must implement
type Plugin interface {
	// Setup is called when the plugin is being setup
	Setup(serviceProvider PluginServiceProvider, config map[string]interface{})

	// ModifyUrl is called just before the URL is being matched against the browser-rules
	// The plugin can modify the URL and return it (or otherwise just return the original URL)
	ModifyUrl(url string) string

	// Shutdown is called when the application is about to exit.
	// Plugins should use this to finish any background work (e.g. writing files).
	// The context carries a deadline — plugins must return before it expires.
	Shutdown(ctx context.Context)
}

// pluginServiceProvider is a struct that implements the PluginServiceProvider interface, providing services that the plugin might need
type pluginServiceProvider struct {
	logger           *slog.Logger
	Settings         *Settings
	configFolderPath string
}

func (p *pluginServiceProvider) GetLogger() *slog.Logger {
	return p.logger
}

func (p *pluginServiceProvider) GetSettings() *Settings {
	return p.Settings
}

func (p *pluginServiceProvider) GetConfigFolderPath() string {
	return p.configFolderPath
}

func NewPluginServiceProvider(logger *slog.Logger, settings *Settings, configFolderPath string) PluginServiceProvider {
	return &pluginServiceProvider{logger: logger, Settings: settings, configFolderPath: configFolderPath}
}
