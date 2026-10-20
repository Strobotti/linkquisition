//go:build windows

package main

import (
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
