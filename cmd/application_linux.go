//go:build linux

package main

import (
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/freedesktop"
)

func newPlatformServices() (linkquisition.BrowserService, linkquisition.SettingsService) {
	xdgService := &freedesktop.XdgService{}
	desktopEntryService := &freedesktop.DesktopEntryService{}
	browserService := &freedesktop.BrowserService{
		XdgService:          xdgService,
		DesktopEntryService: desktopEntryService,
		BrowserIconLoader: &freedesktop.DefaultBrowserIconLoader{
			XdgService:          xdgService,
			DesktopEntryService: desktopEntryService,
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
