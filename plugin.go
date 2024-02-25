package linkquisition

import "log/slog"

// PluginServiceProvider is an interface that provides the logger and settings to the plugin
// This is passed to the plugin as a dependency when being setup.
type PluginServiceProvider interface {
	GetLogger() *slog.Logger
	GetSettings() *Settings
}

// Plugin is an interface that all plugins must implement
type Plugin interface {
	// Setup is called when the plugin is being setup
	Setup(serviceProvider PluginServiceProvider, config map[string]interface{})

	// ModifyUrl is called just before the URL is being matched against the browser-rules
	// The plugin can modify the URL and return it (or otherwise just return the original URL)
	ModifyUrl(url string) string
}

// pluginServiceProvider is a struct that implements the PluginServiceProvider interface, providing services that the plugin might need
type pluginServiceProvider struct {
	logger   *slog.Logger
	Settings *Settings
}

func (p *pluginServiceProvider) GetLogger() *slog.Logger {
	return p.logger
}

func (p *pluginServiceProvider) GetSettings() *Settings {
	return p.Settings
}

func NewPluginServiceProvider(logger *slog.Logger, settings *Settings) PluginServiceProvider {
	return &pluginServiceProvider{logger: logger, Settings: settings}
}
