package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/freedesktop"
)

type Application struct {
	Fapp            fyne.App
	XdgService      freedesktop.XdgService
	BrowserService  linkquisition.BrowserService
	SettingsService linkquisition.SettingsService
}

func NewApplication() *Application {
	fapp := app.New()

	xdgService := &freedesktop.XdgService{}
	browserService := &freedesktop.BrowserService{
		XdgService:          xdgService,
		DesktopEntryService: &freedesktop.DesktopEntryService{},
	}

	settingsService := &freedesktop.SettingsService{
		BrowserService: browserService,
	}

	return &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
	}
}

func (a *Application) Run(_ context.Context) error {
	args := os.Args
	if len(args) < 2 { //nolint:gomnd
		configurator := NewConfigurator(a.Fapp, a.BrowserService, a.SettingsService)
		return configurator.Run()
	}

	if args[1] == "--version" || args[1] == "-v" || args[1] == "version" {
		fmt.Printf("Version: %s\n", version)
		return nil
	}

	var err error
	urlToOpen := args[1]

	if _, err = url.ParseRequestURI(urlToOpen); err != nil {
		fmt.Printf("Invalid URL: %s\n", urlToOpen)
		fmt.Printf("Usage: %s <url>\n", args[0])

		return nil
	}

	var browsers []linkquisition.Browser

	if a.SettingsService.IsConfigured() {
		settings, settingsErr := a.SettingsService.ReadSettings()
		if settingsErr != nil {
			return fmt.Errorf("error reading settings: %v", settingsErr)
		}
		if browser, matchErr := settings.GetMatchingBrowser(urlToOpen); matchErr == nil {
			fmt.Printf("found a matching browser in settings: %s\n", browser.Name)
			if a.BrowserService.OpenUrlWithBrowser(urlToOpen, browser) == nil {
				return nil
			}
		}
		browsers = settings.GetSelectableBrowsers()
		fmt.Printf("found %d browsers configured to be selected\n", len(browsers))
	} else if browsers, err = a.BrowserService.GetAvailableBrowsers(); err != nil {
		return err
	} else {
		fmt.Printf("browsers not configured, falling back to system settings\n")
	}
	bp := NewBrowserPicker(a.Fapp, a.BrowserService, browsers)
	return bp.Run(context.Background(), urlToOpen)
}
