package main

import (
	"fmt"
	"log/slog"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/resources"
)

// templateKeyName is the template data key used for name substitution in i18n strings.
const templateKeyName = "Name"

type Configurator struct {
	fapp            fyne.App
	browserService  linkquisition.BrowserService
	settingsService linkquisition.SettingsService
	logger          *slog.Logger
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

//nolint:unparam
func (c *Configurator) Run() error {
	w := c.fapp.NewWindow(i18n.T("config.window_title"))

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("config.tab_general"), c.getGeneralTab()),
		container.NewTabItem(i18n.T("config.tab_browsers"), c.getBrowsersTab()),
		container.NewTabItem(i18n.T("config.tab_rules"), c.getRulesTab()),
		container.NewTabItem(i18n.T("config.tab_plugins"), c.getPluginsTab()),
		container.NewTabItem(i18n.T("config.tab_about"), c.getAboutTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)

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
		sections.Add(c.buildNoBrowsersWarning())
		sections.Add(widget.NewSeparator())
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

	hideGuideCheck := widget.NewCheck(i18n.T("config.hide_keyboard_guide"), func(checked bool) {
		s := c.settingsService.GetSettings()
		s.Ui.HideKeyboardGuideLabel = checked
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving UI setting", "error", err)
		}
	})
	hideGuideCheck.Checked = settings.Ui.HideKeyboardGuideLabel

	// Picker layout selector
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

	maxItemsRow := container.NewBorder(
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

	return container.NewVBox(
		hideGuideCheck,
		widget.NewSeparator(),
		container.NewBorder(
			nil, nil,
			widget.NewLabel(i18n.T("config.picker_layout_label")), nil,
			layoutSelect,
		),
		maxItemsRow,
	)
}

func (c *Configurator) getAboutTab() fyne.CanvasObject {
	openURL := func() {
		if err := c.openExternalURL("https://github.com/Strobotti/linkquisition"); err != nil {
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

	githubLink := widget.NewButton("github.com/Strobotti/linkquisition", openURL)
	githubLink.Importance = widget.LowImportance

	details := container.NewVBox(
		container.NewHBox(widget.NewLabel(i18n.T("about.author_label")), widget.NewLabel("Juha Jantunen")),
		container.NewHBox(widget.NewLabel(i18n.T("about.license_label")), widget.NewLabel("MIT")),
		container.NewHBox(widget.NewLabel(i18n.T("about.github_label")), githubLink),
	)

	return container.NewVBox(
		container.NewHBox(icon, title),
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		details,
	)
}

// openExternalURL opens a URL in a real browser, bypassing Linkquisition if it is
// the default browser (which would otherwise cause a circular loop).
func (c *Configurator) openExternalURL(rawURL string) error {
	if !c.browserService.AreWeTheDefaultBrowser() {
		return c.browserService.OpenUrlWithDefaultBrowser(rawURL)
	}

	// We are the default browser, so we need to pick a real browser to open with
	browsers, err := c.browserService.GetAvailableBrowsers()
	if err != nil || len(browsers) == 0 {
		// Last resort: try anyway
		return c.browserService.OpenUrlWithDefaultBrowser(rawURL)
	}

	return c.browserService.OpenUrlWithBrowser(rawURL, &browsers[0])
}
