package linkquisition

import "log/slog"

type PluginServiceProvider interface {
	GetLogger() *slog.Logger
	GetSettings() *Settings
}

type Plugin interface {
	Setup(serviceProvider PluginServiceProvider, config map[string]interface{})
	ModifyUrl(url string) string
}

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
