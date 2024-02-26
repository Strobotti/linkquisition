package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/freedesktop"
)

const logDirPerms = 0755
const logFilePerms = 0644

type Application struct {
	Fapp            fyne.App
	XdgService      freedesktop.XdgService
	BrowserService  linkquisition.BrowserService
	SettingsService linkquisition.SettingsService

	Logger  *slog.Logger
	plugins []linkquisition.Plugin
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

	logger := setupLogger(settingsService)

	pluginServiceProvider := linkquisition.NewPluginServiceProvider(logger, settingsService.GetSettings())

	a := &Application{
		Fapp:            fapp,
		BrowserService:  browserService,
		SettingsService: settingsService,
		Logger:          logger,
		plugins:         setupPlugins(settingsService, pluginServiceProvider, logger),
	}

	return a
}

func setupPlugins(
	settingsService linkquisition.SettingsService,
	pluginServiceProvider linkquisition.PluginServiceProvider,
	logger *slog.Logger,
) []linkquisition.Plugin {
	settings := settingsService.GetSettings()
	var plugins []linkquisition.Plugin

	for _, pluginSettings := range settings.Plugins {
		pluginPath := pluginSettings.Path
		if !strings.HasSuffix(pluginPath, ".so") {
			pluginPath += ".so"
		}

		if _, err := os.Stat(pluginPath); err != nil {
			pluginPathToCheck := filepath.Join(settingsService.GetPluginFolderPath(), pluginPath)
			if _, err := os.Stat(pluginPathToCheck); err == nil {
				pluginPath = pluginPathToCheck
			} else {
				logger.Error("Error loading plugin", "plugin", pluginSettings.Path, "error", err.Error())
				continue
			}
		}

		plug, err := plugin.Open(pluginPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		symbol, err := plug.Lookup("Plugin")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		var p linkquisition.Plugin
		p, ok := symbol.(linkquisition.Plugin)
		if !ok {
			fmt.Printf("unexpected type from module symbol: %T\n", symbol)
		} else {
			p.Setup(pluginServiceProvider, pluginSettings.Settings)
			plugins = append(plugins, p)
		}
	}

	return plugins
}

func setupLogger(settingsService linkquisition.SettingsService) *slog.Logger {
	fallbackLog := slog.New(slog.NewTextHandler(os.Stdout, nil))
	settings := settingsService.GetSettings()

	// ensure the path to the log file exists
	if err := os.MkdirAll(settingsService.GetLogFolderPath(), logDirPerms); err != nil {
		fmt.Printf("error creating log folder: %v\n", err)

		return fallbackLog
	}

	var logWriter io.Writer
	var err error

	if logWriter, err = os.OpenFile(settingsService.GetLogFilePath(), os.O_WRONLY|os.O_CREATE|os.O_APPEND, logFilePerms); err != nil {
		fmt.Printf("error opening log file for writing: %v\n", err)

		return fallbackLog
	}

	logHandlerOpts := &slog.HandlerOptions{
		Level: linkquisition.MapSettingsLogLevelToSlog(
			settings.LogLevel,
		),
	}

	return slog.New(slog.NewTextHandler(logWriter, logHandlerOpts))
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

	var err error

	a.Logger.Debug(fmt.Sprintf("Starting linkquisition with args: `%s`", strings.Join(os.Args, " ")))

	urlToOpen := args[1]

	if _, err = url.ParseRequestURI(urlToOpen); err != nil {
		a.Logger.Error("Invalid URL: " + urlToOpen)

		return nil
	}

	for _, plug := range a.plugins {
		urlToOpen = plug.ModifyUrl(urlToOpen)
	}

	isConfigured, configErr := a.SettingsService.IsConfigured()
	if configErr != nil {
		a.Logger.Warn("configuration error", "error", configErr.Error())
	}

	if isConfigured {
		if browser, matchErr := a.SettingsService.GetSettings().GetMatchingBrowser(urlToOpen); matchErr == nil {
			a.Logger.Debug(fmt.Sprintf("found a matching browser-rule for browser `%s` with URL `%s`", browser.Name, urlToOpen))
			if a.BrowserService.OpenUrlWithBrowser(urlToOpen, browser) == nil {
				return nil
			}
		}
		browsers = a.SettingsService.GetSettings().GetSelectableBrowsers()
	} else if browsers, err = a.BrowserService.GetAvailableBrowsers(); err != nil {
		return err
	} else {
		a.Logger.Warn("browsers not configured, falling back to system settings")
	}
	bp := NewBrowserPicker(a.Fapp, a.BrowserService, browsers)
	return bp.Run(context.Background(), urlToOpen)
}
