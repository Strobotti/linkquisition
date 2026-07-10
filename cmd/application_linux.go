//go:build linux

package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/freedesktop"
)

func newPlatformServices() (linkquisition.BrowserService, linkquisition.SettingsService) {
	xdgService := &freedesktop.XdgService{}
	browserService := &freedesktop.BrowserService{
		XdgService:          xdgService,
		DesktopEntryService: &freedesktop.DesktopEntryService{},
		BrowserIconLoader: &freedesktop.DefaultBrowserIconLoader{
			XdgService:          xdgService,
			DesktopEntryService: &freedesktop.DesktopEntryService{},
		},
	}

	settingsService := &linkquisition.FileSettingsService{
		BrowserService: browserService,
		PathProvider:   &freedesktop.PathProvider{},
	}

	return browserService, settingsService
}

// getURLFromPlatformEvent on Linux always returns empty — URLs come via argv.
func getURLFromPlatformEvent() string {
	return ""
}

func NewApplication() *Application {
	fapp := app.New()
	browserService, settingsService := newPlatformServices()

	applyTheme(fapp, settingsService)

	logger := setupLogger(settingsService)
	pluginServiceProvider := linkquisition.NewPluginServiceProvider(
		logger, settingsService.GetSettings(), settingsService.GetConfigFolderPath(),
	)

	return &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
		Logger:          logger,
		plugins:         setupPlugins(settingsService, pluginServiceProvider, logger),
	}
}
