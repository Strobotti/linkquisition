package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/ui"
	"github.com/strobotti/linkquisition/internal/updater"
	"github.com/strobotti/linkquisition/resources"
)

// templateKeyName is the template data key used for name substitution in i18n strings.
const templateKeyName = "Name"

// osDarwin is the runtime.GOOS value for macOS, extracted to satisfy goconst.
const osDarwin = "darwin"

type Configurator struct {
	fapp            fyne.App
	browserService  linkquisition.BrowserService
	settingsService linkquisition.SettingsService
	logger          *slog.Logger

	// noBrowsersWarning holds the warning container shown on the General tab
	// when no browsers are configured. It is hidden after a successful scan.
	noBrowsersWarning fyne.CanvasObject
}

func NewConfigurator(
	fapp fyne.App,
	browserService linkquisition.BrowserService,
	settingsService linkquisition.SettingsService,
	logger *slog.Logger,
) *Configurator {
	return &Configurator{
		fapp:            fapp,
		browserService:  browserService,
		settingsService: settingsService,
		logger:          logger,
	}
}

// parentWindow returns the first open window, typically the configurator's main window.
// Returns nil if no windows are open (should not happen in practice).
func (c *Configurator) parentWindow() fyne.Window {
	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return nil
	}
	return windows[0]
}

//nolint:unparam
func (c *Configurator) Run() error {
	w := c.fapp.NewWindow(i18n.T("config.window_title"))

	tabItems := []*container.TabItem{
		container.NewTabItem(i18n.T("config.tab_general"), c.getGeneralTab()),
		container.NewTabItem(i18n.T("config.tab_browsers"), c.getBrowsersTab()),
		container.NewTabItem(i18n.T("config.tab_rules"), c.getRulesTab()),
	}

	// Plugins tab is only shown on platforms that support plugins
	if pluginsContent := c.getPluginsTab(); pluginsContent != nil {
		tabItems = append(tabItems, container.NewTabItem(i18n.T("config.tab_plugins"), pluginsContent))
	}

	tabItems = append(tabItems,
		container.NewTabItem(i18n.T("config.tab_security"), c.getSecurityTab(w)),
		container.NewTabItem(i18n.T("config.tab_about"), c.getAboutTab(w)),
	)

	tabs := container.NewAppTabs(tabItems...)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)

	w.Canvas().SetOnTypedKey(func(keyEvent *fyne.KeyEvent) {
		if keyEvent.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.Resize(fyne.NewSize(700, 600)) //nolint:mnd
	w.CenterOnScreen()

	w.ShowAndRun()

	return nil
}

func (c *Configurator) getGeneralTab() fyne.CanvasObject {
	sections := container.NewVBox()

	// Only show "make default" section if NOT already the default browser
	if !c.browserService.AreWeTheDefaultBrowser() {
		sections.Add(c.buildMakeDefaultSection())
		sections.Add(widget.NewSeparator())
	}

	// Onboarding: show a warning if no browsers are configured
	if settings := c.settingsService.GetSettings(); len(settings.Browsers) == 0 {
		c.noBrowsersWarning = container.NewVBox(c.buildNoBrowsersWarning(), widget.NewSeparator())
		sections.Add(c.noBrowsersWarning)
	}

	sections.Add(c.buildLanguageSection())
	sections.Add(widget.NewSeparator())
	sections.Add(c.buildLogLevelSection())
	sections.Add(widget.NewSeparator())
	sections.Add(c.buildUiSection())

	// Platform-specific note (e.g. macOS picker limitation) anchored to bottom
	if note := c.buildPlatformNote(); note != nil {
		sections.Add(layout.NewSpacer())
		sections.Add(widget.NewSeparator())
		sections.Add(note)
	}

	return sections
}

func (c *Configurator) buildNoBrowsersWarning() fyne.CanvasObject {
	warningLabel := widget.NewLabel(i18n.T("config.no_browsers_warning"))
	warningLabel.Wrapping = fyne.TextWrapWord
	warningLabel.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewVBox(warningLabel)
}

func (c *Configurator) hideNoBrowsersWarning() {
	if c.noBrowsersWarning != nil {
		c.noBrowsersWarning.Hide()
		c.noBrowsersWarning = nil
	}
}

func (c *Configurator) buildMakeDefaultSection() fyne.CanvasObject {
	makeDefaultLabel := widget.NewLabel(i18n.T("config.make_default_label"))

	setupButton := func(button *widget.Button, isDefault bool) {
		if isDefault {
			button.SetText(i18n.T("config.make_default_done"))
			button.Disable()
		} else {
			button.SetText(i18n.T("config.make_default_button"))
			button.Enable()
		}
	}

	makeDefaultButton := widget.NewButton(i18n.T("config.make_default_checking"), func() {})
	makeDefaultButton.OnTapped = func() {
		makeDefaultButton.Disable()
		err := c.browserService.MakeUsTheDefaultBrowser()
		if err != nil {
			makeDefaultButton.SetText(i18n.T("config.make_default_error"))
			makeDefaultButton.Enable()
			c.logger.Error("Error making Linkquisition the default browser", "error", err)
		} else {
			setupButton(makeDefaultButton, true)
		}
	}
	makeDefaultButton.Disable()

	setupButton(makeDefaultButton, c.browserService.AreWeTheDefaultBrowser())

	return container.NewVBox(makeDefaultLabel, makeDefaultButton)
}

func (c *Configurator) buildLanguageSection() fyne.CanvasObject {
	locales := i18n.AvailableLocales()
	autoLabel := i18n.T("config.language_auto")

	// Sort locales alphabetically by display name.
	sort.Slice(locales, func(i, j int) bool {
		return i18n.LocaleDisplayName(locales[i]) < i18n.LocaleDisplayName(locales[j])
	})

	options := []string{autoLabel}
	for _, code := range locales {
		options = append(options, fmt.Sprintf("%s (%s)", i18n.LocaleDisplayName(code), code))
	}

	currentLocale := c.settingsService.GetSettings().Locale
	selectedOption := autoLabel
	for _, code := range locales {
		if code == currentLocale {
			selectedOption = fmt.Sprintf("%s (%s)", i18n.LocaleDisplayName(code), code)
			break
		}
	}

	languageSelect := widget.NewSelect(options, func(selected string) {
		newLocale := ""
		if selected != autoLabel {
			for _, code := range locales {
				if selected == fmt.Sprintf("%s (%s)", i18n.LocaleDisplayName(code), code) {
					newLocale = code
					break
				}
			}
		}

		settings := c.settingsService.GetSettings()
		settings.Locale = newLocale
		if err := c.settingsService.WriteSettings(settings); err != nil {
			c.logger.Error("Error saving locale setting", "error", err)
		}
	})
	languageSelect.Selected = selectedOption

	restartNote := widget.NewLabel(i18n.T("config.language_restart_note"))
	restartNote.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel(i18n.T("config.language_label")), nil, languageSelect),
		restartNote,
	)
}

func (c *Configurator) buildLogLevelSection() fyne.CanvasObject {
	levels := []string{
		linkquisition.LogLevelDebug,
		linkquisition.LogLevelInfo,
		linkquisition.LogLevelWarn,
		linkquisition.LogLevelError,
	}
	currentLevel := c.settingsService.GetSettings().LogLevel
	if currentLevel == "" {
		currentLevel = linkquisition.LogLevelWarn
	}

	logLevelSelect := widget.NewSelect(levels, func(selected string) {
		settings := c.settingsService.GetSettings()
		settings.LogLevel = selected
		if err := c.settingsService.WriteSettings(settings); err != nil {
			c.logger.Error("Error saving log level setting", "error", err)
		}
	})
	logLevelSelect.Selected = currentLevel

	return container.NewBorder(nil, nil, widget.NewLabel(i18n.T("config.log_level_label")), nil, logLevelSelect)
}

func (c *Configurator) buildUiSection() fyne.CanvasObject {
	settings := c.settingsService.GetSettings()

	themeRow, themeNote := c.buildThemeSelector(settings)

	hideGuideCheck := widget.NewCheck(i18n.T("config.hide_keyboard_guide"), func(checked bool) {
		s := c.settingsService.GetSettings()
		s.Ui.HideKeyboardGuideLabel = checked
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving UI setting", "error", err)
		}
	})
	hideGuideCheck.Checked = settings.Ui.HideKeyboardGuideLabel

	layoutRow, maxItemsRow := c.buildPickerLayoutSelector(settings)

	faviconSection := c.buildFaviconSection(settings)

	return container.NewVBox(
		themeRow,
		themeNote,
		widget.NewSeparator(),
		hideGuideCheck,
		widget.NewSeparator(),
		layoutRow,
		maxItemsRow,
		widget.NewSeparator(),
		faviconSection,
	)
}

func (c *Configurator) buildThemeSelector(settings *linkquisition.Settings) (row, note fyne.CanvasObject) {
	themeOptions := []string{
		i18n.T("config.theme_system"),
		i18n.T("config.theme_dark"),
		i18n.T("config.theme_light"),
	}

	currentTheme := settings.Ui.GetTheme()
	selectedTheme := themeOptions[0]
	switch currentTheme {
	case linkquisition.ThemeDark:
		selectedTheme = themeOptions[1]
	case linkquisition.ThemeLight:
		selectedTheme = themeOptions[2]
	}

	themeSelect := widget.NewSelect(themeOptions, func(selected string) {
		s := c.settingsService.GetSettings()
		switch selected {
		case themeOptions[1]:
			s.Ui.Theme = linkquisition.ThemeDark
		case themeOptions[2]:
			s.Ui.Theme = linkquisition.ThemeLight
		default:
			s.Ui.Theme = linkquisition.ThemeSystem
		}
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving theme setting", "error", err)
		}
	})
	themeSelect.Selected = selectedTheme

	restartLabel := widget.NewLabel(i18n.T("config.theme_restart_note"))
	restartLabel.TextStyle = fyne.TextStyle{Italic: true}

	row = container.NewBorder(
		nil, nil,
		widget.NewLabel(i18n.T("config.theme_label")), nil,
		themeSelect,
	)

	return row, restartLabel
}

func (c *Configurator) buildPickerLayoutSelector(settings *linkquisition.Settings) (layoutRow, maxItemsRow fyne.CanvasObject) {
	layoutOptions := []string{
		i18n.T("config.picker_layout_vertical"),
		i18n.T("config.picker_layout_horizontal"),
	}

	currentLayout := settings.Ui.GetPickerLayout()
	selectedLayout := layoutOptions[0]
	if currentLayout == linkquisition.PickerLayoutHorizontal {
		selectedLayout = layoutOptions[1]
	}

	// Max items per row entry (only relevant for horizontal layout)
	maxItemsOptions := []string{"3", "4", "5", "6"}
	currentMaxItems := fmt.Sprintf("%d", settings.Ui.GetMaxItemsPerRow())
	// Clamp to valid range for display
	if settings.Ui.GetMaxItemsPerRow() < 3 { //nolint:mnd
		currentMaxItems = "3"
	} else if settings.Ui.GetMaxItemsPerRow() > 6 { //nolint:mnd
		currentMaxItems = "6"
	}

	maxItemsSelect := widget.NewSelect(maxItemsOptions, func(value string) {
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return
		}
		s := c.settingsService.GetSettings()
		s.Ui.MaxItemsPerRow = n
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving max items per row setting", "error", err)
		}
	})
	maxItemsSelect.Selected = currentMaxItems

	maxItemsRow = container.NewBorder(
		nil, nil,
		widget.NewLabel(i18n.T("config.picker_max_per_row_label")), nil,
		maxItemsSelect,
	)

	updateMaxItemsVisible := func(isHorizontal bool) {
		if isHorizontal {
			maxItemsRow.Show()
		} else {
			maxItemsRow.Hide()
		}
	}
	updateMaxItemsVisible(currentLayout == linkquisition.PickerLayoutHorizontal)

	layoutSelect := widget.NewSelect(layoutOptions, func(selected string) {
		s := c.settingsService.GetSettings()
		if selected == layoutOptions[1] {
			s.Ui.PickerLayout = linkquisition.PickerLayoutHorizontal
			updateMaxItemsVisible(true)
		} else {
			s.Ui.PickerLayout = linkquisition.PickerLayoutVertical
			updateMaxItemsVisible(false)
		}
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving picker layout setting", "error", err)
		}
	})
	layoutSelect.Selected = selectedLayout

	layoutRow = container.NewBorder(
		nil, nil,
		widget.NewLabel(i18n.T("config.picker_layout_label")), nil,
		layoutSelect,
	)

	return layoutRow, maxItemsRow
}

func (c *Configurator) buildFaviconSection(settings *linkquisition.Settings) fyne.CanvasObject {
	strategyOptions := []string{
		i18n.T("config.favicon_strategy_direct"),
		i18n.T("config.favicon_strategy_parsed"),
		i18n.T("config.favicon_strategy_google"),
	}

	currentStrategy := settings.Ui.GetFaviconStrategy()
	selectedStrategy := strategyOptions[0]
	switch currentStrategy {
	case linkquisition.FaviconStrategyParsed:
		selectedStrategy = strategyOptions[1]
	case linkquisition.FaviconStrategyGoogle:
		selectedStrategy = strategyOptions[2]
	}

	strategyDesc := widget.NewLabel(i18n.T("config.favicon_strategy_direct_desc"))
	strategyDesc.TextStyle = fyne.TextStyle{Italic: true}
	strategyDesc.Wrapping = fyne.TextWrapWord

	updateDescription := func(strategy string) {
		switch strategy {
		case linkquisition.FaviconStrategyParsed:
			strategyDesc.SetText(i18n.T("config.favicon_strategy_parsed_desc"))
		case linkquisition.FaviconStrategyGoogle:
			strategyDesc.SetText(i18n.T("config.favicon_strategy_google_desc"))
		default:
			strategyDesc.SetText(i18n.T("config.favicon_strategy_direct_desc"))
		}
	}
	updateDescription(currentStrategy)

	strategySelect := widget.NewSelect(strategyOptions, func(selected string) {
		s := c.settingsService.GetSettings()
		switch selected {
		case strategyOptions[1]:
			s.Ui.FaviconStrategy = linkquisition.FaviconStrategyParsed
			updateDescription(linkquisition.FaviconStrategyParsed)
		case strategyOptions[2]:
			s.Ui.FaviconStrategy = linkquisition.FaviconStrategyGoogle
			updateDescription(linkquisition.FaviconStrategyGoogle)
		default:
			s.Ui.FaviconStrategy = linkquisition.FaviconStrategyDirect
			updateDescription(linkquisition.FaviconStrategyDirect)
		}
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving favicon strategy setting", "error", err)
		}
	})
	strategySelect.Selected = selectedStrategy

	strategyRow := container.NewBorder(
		nil, nil,
		widget.NewLabel(i18n.T("config.favicon_strategy_label")), nil,
		strategySelect,
	)

	// Clear cache button
	clearCacheButton := widget.NewButton(i18n.T("config.favicon_clear_cache"), func() {})
	clearCacheButton.Importance = widget.LowImportance
	clearCacheButton.OnTapped = func() {
		cacheDir := filepath.Join(c.settingsService.GetConfigFolderPath(), "favicons")
		if err := os.RemoveAll(cacheDir); err != nil {
			c.logger.Error("Error clearing favicon cache", "error", err)
			clearCacheButton.SetText(i18n.T("config.favicon_clear_cache_error"))
		} else {
			clearCacheButton.SetText(i18n.T("config.favicon_clear_cache_done"))
			clearCacheButton.Disable()
		}
	}

	// Show strategy options and clear cache only when favicon is enabled
	updateStrategyVisible := func(enabled bool) {
		if enabled {
			strategyRow.Show()
			strategyDesc.Show()
			clearCacheButton.Show()
		} else {
			strategyRow.Hide()
			strategyDesc.Hide()
			clearCacheButton.Hide()
		}
	}
	updateStrategyVisible(settings.Ui.ShowFavicon)

	faviconCheck := widget.NewCheck(i18n.T("config.favicon_label"), func(checked bool) {
		s := c.settingsService.GetSettings()
		s.Ui.ShowFavicon = checked
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving favicon setting", "error", err)
		}
		updateStrategyVisible(checked)
	})
	faviconCheck.Checked = settings.Ui.ShowFavicon

	return container.NewVBox(
		faviconCheck,
		strategyRow,
		strategyDesc,
		clearCacheButton,
	)
}

func (c *Configurator) getAboutTab(w fyne.Window) fyne.CanvasObject {
	const githubURL = "https://github.com/Strobotti/linkquisition"

	openURL := func() {
		if err := c.openExternalURL(githubURL); err != nil {
			c.logger.Error("Error opening URL", "error", err)
		}
	}

	icon := widget.NewButtonWithIcon(
		"",
		resources.LinkquisitionIcon,
		openURL,
	)

	title := widget.NewLabel(fmt.Sprintf("Linkquisition %s", version))
	title.TextStyle = fyne.TextStyle{Bold: true}

	description := widget.NewLabel(i18n.T("about.description"))
	description.Wrapping = fyne.TextWrapWord

	githubLink := ui.NewLinkWithCopy("github.com/Strobotti/linkquisition", githubURL, w, c.urlOpener())

	details := container.NewVBox(
		container.NewHBox(widget.NewLabel(i18n.T("about.author_label")), widget.NewLabel("Juha Jantunen")),
		container.NewHBox(widget.NewLabel(i18n.T("about.license_label")), widget.NewLabel("MIT")),
		container.NewHBox(widget.NewLabel(i18n.T("about.github_label")), githubLink),
	)

	// Update check section
	updateSection := c.buildUpdateCheckSection(w)

	// Bug reporting section (pinned to bottom)
	bugReportSection := c.buildBugReportSection(w)

	topContent := container.NewVBox(
		container.NewHBox(icon, title),
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		details,
		widget.NewSeparator(),
		updateSection,
	)

	return container.NewBorder(topContent, bugReportSection, nil, nil)
}

func (c *Configurator) buildUpdateCheckSection(w fyne.Window) fyne.CanvasObject {
	statusLabel := widget.NewLabel("")

	releaseLink := container.NewHBox()
	releaseLink.Hide()

	checkButton := widget.NewButton(i18n.T("about.check_updates"), nil)
	checkButton.OnTapped = func() {
		checkButton.Disable()
		checkButton.SetText(i18n.T("about.checking_updates"))
		statusLabel.SetText("")
		releaseLink.Hide()

		go func() {
			result, err := updater.Check(context.Background(), version)

			fyne.Do(func() {
				checkButton.Enable()
				checkButton.SetText(i18n.T("about.check_updates"))

				if err != nil {
					c.logger.Error("Update check failed", "error", err)
					statusLabel.SetText(i18n.T("about.update_error"))
					return
				}

				if result.IsNewer {
					statusLabel.SetText(i18n.T("about.update_available", map[string]interface{}{
						"Version": result.LatestVersion,
					}))

					releaseLink.RemoveAll()
					releaseLink.Add(ui.NewLinkWithCopy(i18n.T("about.view_release"), result.ReleaseURL, w, c.urlOpener()))
					releaseLink.Show()
				} else {
					statusLabel.SetText(i18n.T("about.up_to_date"))
				}
			})
		}()
	}

	return container.NewHBox(checkButton, statusLabel, releaseLink)
}

func (c *Configurator) buildBugReportSection(w fyne.Window) fyne.CanvasObject {
	const issuesURL = "https://github.com/Strobotti/linkquisition/issues"

	titleLabel := widget.NewLabel(i18n.T("about.bug_report_title"))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	description := widget.NewLabel(i18n.T("about.bug_report_description"))
	description.Wrapping = fyne.TextWrapWord

	openLogButton := widget.NewButtonWithIcon(
		i18n.T("about.bug_report_open_log"),
		theme.FileIcon(),
		func() {
			logPath := c.settingsService.GetLogFilePath()
			c.logger.Debug("Opening log file in editor", "path", logPath)

			if err := openFileInEditor(logPath); err != nil {
				c.logger.Error("Failed to open log file", "path", logPath, "error", err)
			}
		},
	)

	issuesLink := ui.NewLinkWithCopy(i18n.T("about.bug_report_issues_link"), issuesURL, w, c.urlOpener())

	return container.NewVBox(
		titleLabel,
		description,
		container.NewHBox(openLogButton, issuesLink),
	)
}

// openExternalURL opens a URL in a real browser, bypassing Linkquisition if it is
// the default browser (which would otherwise cause a circular loop).
func (c *Configurator) openExternalURL(rawURL string) error {
	return openExternalURLWithService(rawURL, c.browserService)
}

// urlOpener returns a URLOpener function for use with UI link widgets.
func (c *Configurator) urlOpener() ui.URLOpener {
	return func(rawURL string) error {
		c.logger.Debug("Opening external URL from configurator", "url", rawURL)

		return c.openExternalURL(rawURL)
	}
}

// openExternalURLWithService opens a URL using the given browser service, choosing
// a real browser if we are the default (to avoid a circular loop).
func openExternalURLWithService(rawURL string, browserService linkquisition.BrowserService) error {
	if !browserService.AreWeTheDefaultBrowser() {
		slog.Debug("Opening URL with system default browser", "url", rawURL)

		return browserService.OpenUrlWithDefaultBrowser(rawURL)
	}

	// We are the default browser, so we need to pick a real browser to open with
	slog.Debug("We are the default browser, picking a real browser to open URL", "url", rawURL)

	browsers, err := browserService.GetAvailableBrowsers()
	if err != nil || len(browsers) == 0 {
		slog.Debug("No browsers available, falling back to default handler", "url", rawURL, "error", err)
		// Last resort: try anyway
		return browserService.OpenUrlWithDefaultBrowser(rawURL)
	}

	slog.Debug("Opening URL with first available browser", "url", rawURL, "browser", browsers[0].Name)

	return browserService.OpenUrlWithBrowser(rawURL, &browsers[0])
}

// openFileInEditor opens a file in the system's default text editor.
func openFileInEditor(path string) error {
	ctx := context.Background()

	switch runtime.GOOS {
	case osDarwin:
		return exec.CommandContext(ctx, "open", "-t", path).Start()
	case "windows":
		return exec.CommandContext(ctx, "cmd", "/c", "start", "", path).Start()
	default: // Linux and other Unix-like systems
		return exec.CommandContext(ctx, "xdg-open", path).Start()
	}
}
