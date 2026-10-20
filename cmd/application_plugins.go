//go:build !windows

package main

import (
	"log/slog"

	"github.com/strobotti/linkquisition"
)

func loadApplicationPlugins(
	settingsService linkquisition.SettingsService,
	logger *slog.Logger,
) []linkquisition.Plugin {
	if noPlugins {
		return nil
	}

	pluginServiceProvider := linkquisition.NewPluginServiceProvider(
		logger,
		settingsService.GetSettings(),
		settingsService.GetConfigFolderPath(),
	)
	return setupPlugins(settingsService, pluginServiceProvider, logger, parsePluginOpts(pluginOpts))
}
