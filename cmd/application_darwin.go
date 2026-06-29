//go:build darwin

package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/darwin"
)

func newPlatformServices() (linkquisition.BrowserService, linkquisition.SettingsService) {
	browserService := &darwin.BrowserService{}

	settingsService := &linkquisition.FileSettingsService{
		BrowserService: browserService,
		PathProvider:   &darwin.PathProvider{},
	}

	return browserService, settingsService
}

// getURLFromPlatformEvent registers the Apple Event handler, briefly pumps the macOS
// run loop to receive any pending URL event, and returns it (or empty string).
func getURLFromPlatformEvent() string {
	darwin.RegisterURLHandler()

	// Pump the run loop to allow the Apple Event to be delivered.
	darwin.PumpEvents(0.5)

	// Check if a URL arrived
	select {
	case u := <-darwin.URLChannel:
		return u
	default:
		return ""
	}
}

func NewApplication() *Application {
	fapp := app.New()
	browserService, settingsService := newPlatformServices()

	logger := setupLogger(settingsService)
	pluginServiceProvider := linkquisition.NewPluginServiceProvider(logger, settingsService.GetSettings())

	return &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
		Logger:          logger,
		plugins:         setupPlugins(settingsService, pluginServiceProvider, logger),
	}
}
