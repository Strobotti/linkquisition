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
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

const logDirPerms = 0755
const logFilePerms = 0644
const pluginShutdownTimeout = 10 * time.Second

type Application struct {
	Fapp            fyne.App
	BrowserService  linkquisition.BrowserService
	SettingsService linkquisition.SettingsService

	Logger  *slog.Logger
	plugins []linkquisition.Plugin
}

func setupPlugins(
	settingsService linkquisition.SettingsService,
	pluginServiceProvider linkquisition.PluginServiceProvider,
	logger *slog.Logger,
) []linkquisition.Plugin {
	settings := settingsService.GetSettings()
	var plugins []linkquisition.Plugin

	for _, pluginSettings := range settings.Plugins {
		if pluginSettings.IsDisabled {
			logger.Debug("Plugin is disabled by configuration directive", "plugin", pluginSettings.Path)
			continue
		}

		pluginPath := pluginSettings.Path
		if !strings.HasSuffix(pluginPath, ".so") {
			pluginPath += ".so"
		}

		if _, err := os.Stat(pluginPath); err != nil {
			pluginPathToCheck := filepath.Join(settingsService.GetPluginFolderPath(), pluginPath)
			if _, err := os.Stat(pluginPathToCheck); err == nil {
				pluginPath = pluginPathToCheck
			} else {
				logger.Error("Error loading plugin: file not found", "plugin", pluginSettings.Path, "checked", pluginPathToCheck, "error", err.Error())
				continue
			}
		}

		plug, err := openPlugin(pluginPath, logger)
		if plug == nil || err != nil {
			logger.Error("Error opening plugin", "plugin", pluginSettings.Path, "path", pluginPath, "error", err.Error())
			continue
		}

		if p, err := setupPlugin(plug, pluginSettings.Settings, pluginServiceProvider); err != nil {
			logger.Error("Error setting up plugin", "plugin", pluginSettings.Path, "error", err.Error())
		} else {
			logger.Debug("Plugin loaded successfully", "plugin", pluginSettings.Path)
			plugins = append(plugins, p)
		}
	}

	return plugins
}

// openPlugin wraps plugin.Open with panic recovery — Go plugin loading can panic
// on interface mismatches or incompatible builds rather than returning an error.
func openPlugin(path string, logger *slog.Logger) (p *plugin.Plugin, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while opening plugin %s: %v", path, r)
			logger.Error("Plugin open panicked", "path", path, "panic", fmt.Sprintf("%v", r))
		}
	}()

	p, err = plugin.Open(path)
	return p, err
}

func setupPlugin(
	plug *plugin.Plugin,
	settings map[string]interface{},
	pluginServiceProvider linkquisition.PluginServiceProvider,
) (
	p linkquisition.Plugin,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while setting up plugin: %v", r)
		}
	}()

	var symbol plugin.Symbol
	symbol, err = plug.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin symbol lookup returned an error: %v", err)
	}

	var ok bool
	p, ok = symbol.(linkquisition.Plugin)
	if !ok {
		return nil, fmt.Errorf(
			"plugin symbol does not implement linkquisition.Plugin interface (got %T) — "+
				"this usually means the plugin was built against a different version of the app",
			symbol,
		)
	} else {
		p.Setup(pluginServiceProvider, settings)
	}

	return p, nil
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
	defer a.shutdownPlugins()

	// Initialize localization before any UI strings are used
	i18n.Init(a.SettingsService.GetSettings().Locale)

	args := os.Args

	urlToOpen := ""

	if len(args) >= 2 { //nolint:mnd
		if args[1] == "--version" || args[1] == "-v" || args[1] == "version" {
			fmt.Printf("Version: %s\n", version)
			return nil
		}
		urlToOpen = args[1]
	} else {
		// No CLI args — check if a URL arrived via platform event (macOS Apple Events)
		urlToOpen = getURLFromPlatformEvent()
	}

	if urlToOpen == "" {
		configurator := NewConfigurator(a.Fapp, a.BrowserService, a.SettingsService, a.Logger)
		return configurator.Run()
	}

	if _, err := url.ParseRequestURI(urlToOpen); err != nil {
		a.Logger.Error("Invalid URL: " + urlToOpen)
		return nil
	}

	a.Logger.Debug(fmt.Sprintf("Starting linkquisition with URL: `%s`", urlToOpen))

	for _, plug := range a.plugins {
		urlToOpen = plug.ModifyUrl(urlToOpen)
	}

	var browsers []linkquisition.Browser

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
	} else {
		var err error
		if browsers, err = a.BrowserService.GetAvailableBrowsers(); err != nil {
			return err
		}
		a.Logger.Warn("browsers not configured, falling back to system settings")
	}

	bp := NewBrowserPicker(a.Fapp, a.BrowserService, browsers, a.SettingsService, a.Logger)
	return bp.Run(context.Background(), urlToOpen)
}

func (a *Application) shutdownPlugins() {
	if len(a.plugins) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginShutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, plug := range a.plugins {
		wg.Add(1)
		go func(p linkquisition.Plugin) {
			defer wg.Done()
			p.Shutdown(ctx)
		}(plug)
	}
	wg.Wait()
}
