package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

const logDirPerms = 0755
const logFilePerms = 0644
const pluginShutdownTimeout = 10 * time.Second
const maxLogFileSize = 1 << 20 // 1 MB
const pluginProcessTimeout = 30 * time.Second
const pluginExtension = ".so"

type Application struct {
	Fapp            fyne.App
	BrowserService  linkquisition.BrowserService
	SettingsService linkquisition.SettingsService

	Logger  *slog.Logger
	plugins []linkquisition.Plugin
}

// applyTheme configures the Fyne app theme based on the user's ui.theme setting.
// "system" (or empty) uses the OS default; "dark" and "light" force the variant.
func applyTheme(fapp fyne.App, settingsService linkquisition.SettingsService) {
	settings := settingsService.GetSettings()

	switch settings.Ui.GetTheme() {
	case linkquisition.ThemeDark:
		fapp.Settings().SetTheme(theme.DarkTheme())
	case linkquisition.ThemeLight:
		fapp.Settings().SetTheme(theme.LightTheme())
	case linkquisition.ThemeSystem:
		// Fyne follows the OS by default — no action needed.
	}
}

func setupPlugins(
	settingsService linkquisition.SettingsService,
	pluginServiceProvider linkquisition.PluginServiceProvider,
	logger *slog.Logger,
	overrides map[string]map[string]string,
) []linkquisition.Plugin {
	settings := settingsService.GetSettings()
	var plugins []linkquisition.Plugin

	for _, pluginSettings := range settings.Plugins {
		if pluginSettings.IsDisabled {
			logger.Debug("Plugin is disabled by configuration directive", "plugin", pluginSettings.Path)
			continue
		}

		pluginPath := pluginSettings.Path
		if !strings.HasSuffix(pluginPath, pluginExtension) {
			pluginPath += pluginExtension
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

		// Apply runtime overrides from --plugin-opt flags
		effectiveSettings := pluginSettings.Settings
		pluginName := strings.TrimSuffix(filepath.Base(pluginSettings.Path), pluginExtension)
		if opts, ok := overrides[pluginName]; ok {
			if effectiveSettings == nil {
				effectiveSettings = make(map[string]interface{})
			}
			for k, v := range opts {
				effectiveSettings[k] = v
				logger.Debug("Plugin setting overridden via --plugin-opt", "plugin", pluginName, "key", k, "value", v)
			}
		}

		if p, err := setupPlugin(plug, effectiveSettings, pluginServiceProvider); err != nil {
			logger.Error("Error setting up plugin", "plugin", pluginSettings.Path, "error", err.Error())
		} else {
			logger.Debug("Plugin loaded successfully", "plugin", pluginSettings.Path)
			plugins = append(plugins, p)
		}
	}

	return plugins
}

// parsePluginOpts parses --plugin-opt flag values ("plugin.key=value") into
// a nested map: map[pluginName]map[settingKey]value.
// Entries that don't match the expected format are silently ignored.
func parsePluginOpts(opts []string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, opt := range opts {
		// Split on first "=" to get "plugin.key" and "value"
		eqIdx := strings.IndexByte(opt, '=')
		if eqIdx < 0 {
			continue
		}
		path := opt[:eqIdx]
		value := opt[eqIdx+1:]

		// Split path on first "." to get plugin name and key
		dotIdx := strings.IndexByte(path, '.')
		if dotIdx < 0 || dotIdx == 0 || dotIdx == len(path)-1 {
			continue
		}
		pluginName := path[:dotIdx]
		key := path[dotIdx+1:]

		if result[pluginName] == nil {
			result[pluginName] = make(map[string]string)
		}
		result[pluginName][key] = value
	}
	return result
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
	}

	if setupErr := p.Setup(pluginServiceProvider, settings); setupErr != nil {
		return nil, fmt.Errorf("plugin setup failed: %w", setupErr)
	}

	return p, nil
}

// probePluginMetadata opens a plugin .so file and retrieves its Metadata without calling Setup.
// Used by the configurator to display plugin info without fully initializing plugins.
func probePluginMetadata(path string, logger *slog.Logger) (meta linkquisition.PluginMetadata, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while probing plugin %s: %v", path, r)
		}
	}()

	plug, openErr := openPlugin(path, logger)
	if plug == nil || openErr != nil {
		return meta, fmt.Errorf("failed to open plugin: %w", openErr)
	}

	symbol, lookupErr := plug.Lookup("Plugin")
	if lookupErr != nil {
		return meta, fmt.Errorf("plugin symbol lookup failed: %w", lookupErr)
	}

	p, ok := symbol.(linkquisition.Plugin)
	if !ok {
		return meta, fmt.Errorf("plugin does not implement Plugin interface (got %T)", symbol)
	}

	return p.Metadata(), nil
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

	logLevel := settings.LogLevel
	if logLevelOverride != "" {
		logLevel = logLevelOverride
	}

	logHandlerOpts := &slog.HandlerOptions{
		Level: linkquisition.MapSettingsLogLevelToSlog(
			logLevel,
		),
	}

	return slog.New(slog.NewTextHandler(logWriter, logHandlerOpts))
}

// rotateLogFile checks the log file size and rotates it if it exceeds the threshold.
// Keeps at most one backup (linkquisition.log.1).
func rotateLogFile(settingsService linkquisition.SettingsService) {
	logPath := settingsService.GetLogFilePath()

	info, err := os.Stat(logPath)
	if err != nil {
		return // file doesn't exist yet, nothing to rotate
	}

	if info.Size() < maxLogFileSize {
		return
	}

	backupPath := logPath + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(logPath, backupPath)
}

// processPlugins runs the URL through all plugins and returns the final URL and action.
func (a *Application) processPlugins(
	ctx context.Context, urlToOpen string,
) (finalURL string, action linkquisition.PluginAction, message string) {
	for _, plug := range a.plugins {
		result := plug.ProcessURL(ctx, urlToOpen)

		switch result.Action {
		case linkquisition.ActionBlock:
			return result.URL, linkquisition.ActionBlock, result.Message
		case linkquisition.ActionWarn:
			return result.URL, linkquisition.ActionWarn, result.Message
		case linkquisition.ActionOpenDirect:
			return result.URL, linkquisition.ActionOpenDirect, ""
		case linkquisition.ActionContinue:
			urlToOpen = result.URL
		}

		// For non-continue actions, stop the chain unless explicitly told to continue
		if result.Action != linkquisition.ActionContinue && !result.ContinueChain {
			break
		}
	}

	return urlToOpen, linkquisition.ActionContinue, ""
}

// RunGUI launches the GUI mode of the application. If urlToOpen is empty, the
// configurator is shown; otherwise the browser picker handles the URL.
func (a *Application) RunGUI(_ context.Context, urlToOpen string) error {
	defer a.shutdownPlugins()

	// Initialize localization before any UI strings are used
	i18n.Init(a.SettingsService.GetSettings().Locale)

	if urlToOpen == "" {
		// Start watching for URLs arriving via platform events (macOS Apple Events)
		// while the configurator is open.
		watchCtx, watchCancel := context.WithCancel(context.Background())
		a.startURLWatcher(watchCtx)

		configurator := NewConfigurator(a.Fapp, a.BrowserService, a.SettingsService, a.Logger)
		err := configurator.Run()
		watchCancel()

		return err
	}

	// Rotate the log file if it has grown too large. Deferred so it runs as one
	// of the last things before the process exits in the URL-opening path.
	defer rotateLogFile(a.SettingsService)

	a.Logger.Debug(fmt.Sprintf("Starting linkquisition with URL: `%s`", urlToOpen))

	ctx, cancel := context.WithTimeout(context.Background(), pluginProcessTimeout)
	defer cancel()

	processedURL, action, message := a.processPlugins(ctx, urlToOpen)

	switch action {
	case linkquisition.ActionBlock:
		a.showBlockDialog(message)
		return nil
	case linkquisition.ActionWarn:
		a.showWarnDialog(processedURL, message)
		return nil
	case linkquisition.ActionOpenDirect:
		return a.openInFirstBrowser(processedURL)
	case linkquisition.ActionContinue:
		// normal flow: match against browser rules or show picker
	}

	return a.openWithBrowserOrPicker(processedURL)
}

// showBlockDialog displays a blocking dialog and exits without opening any URL.
func (a *Application) showBlockDialog(message string) {
	w := a.Fapp.NewWindow(i18n.T("plugin.blocked_title"))
	w.Resize(fyne.NewSize(500, 300)) //nolint:mnd
	w.CenterOnScreen()

	dialog.ShowInformation(
		i18n.T("plugin.blocked_title"),
		message,
		w,
	)

	w.ShowAndRun()
}

// showWarnDialog displays a warning dialog with "Open anyway" and "Cancel" options.
// Cancel is visually highlighted (primary) and is the default action.
// Enter and Escape both dismiss without opening.
func (a *Application) showWarnDialog(urlToOpen, message string) {
	w := a.Fapp.NewWindow(i18n.T("plugin.warning_title"))
	w.Resize(fyne.NewSize(500, 300)) //nolint:mnd
	w.CenterOnScreen()

	cancelBtn := widget.NewButton(i18n.T("plugin.warn_cancel"), func() {
		a.Fapp.Quit()
	})
	cancelBtn.Importance = widget.HighImportance

	proceedBtn := widget.NewButton(i18n.T("plugin.warn_proceed"), func() {
		_ = a.openInFirstBrowser(urlToOpen)
		a.Fapp.Quit()
	})
	proceedBtn.Importance = widget.DangerImportance

	msgLabel := widget.NewLabel(message)
	msgLabel.Wrapping = fyne.TextWrapWord

	d := dialog.NewCustomWithoutButtons(
		i18n.T("plugin.warning_title"),
		container.NewBorder(
			nil,
			container.NewHBox(layout.NewSpacer(), proceedBtn, cancelBtn),
			nil,
			nil,
			msgLabel,
		),
		w,
	)
	d.Resize(fyne.NewSize(460, 200)) //nolint:mnd
	d.Show()

	// Enter and Escape both cancel (safe default)
	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEscape {
			a.Fapp.Quit()
		}
	})
	w.SetCloseIntercept(func() {
		a.Fapp.Quit()
	})

	w.ShowAndRun()
}

// openWithBrowserOrPicker handles the normal URL opening flow: match rules or show picker.
func (a *Application) openWithBrowserOrPicker(urlToOpen string) error {
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

	bp := NewBrowserPicker(a.Fapp, a.BrowserService, browsers, a.SettingsService, a.Logger, a.collectUIHooks())
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

// collectUIHooks returns all loaded plugins that implement the PluginUIHook interface.
func (a *Application) collectUIHooks() []linkquisition.PluginUIHook {
	var hooks []linkquisition.PluginUIHook
	for _, plug := range a.plugins {
		if hook, ok := plug.(linkquisition.PluginUIHook); ok {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}

func (a *Application) openInFirstBrowser(urlToOpen string) error {
	var browsers []linkquisition.Browser

	if isConfigured, _ := a.SettingsService.IsConfigured(); isConfigured {
		browsers = a.SettingsService.GetSettings().GetSelectableBrowsers()
	} else {
		var err error
		if browsers, err = a.BrowserService.GetAvailableBrowsers(); err != nil {
			return err
		}
	}

	if len(browsers) == 0 {
		a.Logger.Error("no browsers available to open URL", "url", urlToOpen)
		return fmt.Errorf("no browsers available")
	}

	a.Logger.Debug(fmt.Sprintf("opening URL in first browser: %s", browsers[0].Name), "url", urlToOpen)
	return a.BrowserService.OpenUrlWithBrowser(urlToOpen, &browsers[0])
}
