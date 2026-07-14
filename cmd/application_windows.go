//go:build windows

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/windows"
)

func newPlatformServices() (linkquisition.BrowserService, linkquisition.SettingsService) {
	browserService := &windows.BrowserService{}

	settingsService := &linkquisition.FileSettingsService{
		BrowserService: browserService,
		PathProvider:   &windows.PathProvider{},
	}

	return browserService, settingsService
}

// getURLFromPlatformEvent on Windows always returns empty — URLs come via argv.
func getURLFromPlatformEvent() string {
	return ""
}

func NewApplication() *Application {
	app.SetMetadata(fyne.AppMetadata{
		ID:         "com.strobotti.linkquisition",
		Name:       "Linkquisition",
		Migrations: map[string]bool{"fyneDo": true},
	})

	fapp := app.NewWithID("com.strobotti.linkquisition")
	browserService, settingsService := newPlatformServices()

	applyTheme(fapp, settingsService)

	logger := setupLogger(settingsService)

	// Plugins are not supported on Windows (Go's plugin package doesn't support Windows).
	// The application starts with an empty plugin list.

	return &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
		Logger:          logger,
		plugins:         nil,
	}
}
