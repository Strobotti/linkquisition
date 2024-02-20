package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"

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
	Logger          *slog.Logger
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

	a := &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
	}

	// TODO we need to pass the logger to the browser service, but this is hacky
	browserService.App = a

	return a
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

	var browsers []linkquisition.Browser

	var logWriter io.Writer
	var settings *linkquisition.Settings
	var err error
	isConfigured := a.SettingsService.IsConfigured()

	// ensure the path to the log file exists
	if err := os.MkdirAll(a.SettingsService.GetLogFolderPath(), 0755); err != nil {
		return fmt.Errorf("error creating log folder: %v", err)
	}

	if logFile, err := os.OpenFile(a.SettingsService.GetLogFilePath(), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
		panic(fmt.Sprintf("error opening log file: %v", err))
	} else {
		defer logFile.Close()
		logWriter = logFile
	}

	if isConfigured {
		if settings, err = a.SettingsService.ReadSettings(); err != nil {
			// TODO: probably should just log this and continue with default settings
			return fmt.Errorf("error reading settings: %v", err)
		}

		logHandlerOpts := &slog.HandlerOptions{
			Level: linkquisition.MapSettingsLogLevelToSlog(
				settings.LogLevel,
			),
		}

		a.Logger = slog.New(slog.NewTextHandler(logWriter, logHandlerOpts))
	} else {
		a.Logger = slog.New(slog.NewTextHandler(logWriter, nil))
	}

	a.Logger.Info(fmt.Sprintf("Starting linkquisition with args: `%s`", strings.Join(os.Args, " ")))

	urlToOpen := args[1]

	if _, err = url.ParseRequestURI(urlToOpen); err != nil {
		// fmt.Printf("Invalid URL: %s\n", urlToOpen)
		// fmt.Printf("Usage: %s <url>\n", args[0])
		a.Logger.Error("Invalid URL: " + urlToOpen)

		return nil
	}

	if isConfigured {
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

func (a *Application) GetLogger() *slog.Logger {
	return a.Logger
}
