//go:build darwin

package main

import (
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/darwin"
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
	darwin.PumpEvents(0.5)

	select {
	case url := <-darwin.URLChannel:
		return url
	default:
		return ""
	}
}
