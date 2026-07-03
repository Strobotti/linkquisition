package main

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/resources"
)

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
		container.NewTabItem(i18n.T("config.tab_rules"), c.getRulesTab()),
		container.NewTabItem(i18n.T("config.tab_about"), c.getAboutTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(500, 400)) //nolint:mnd
	w.CenterOnScreen()

	w.ShowAndRun()

	return nil
}

func (c *Configurator) getGeneralTab() fyne.CanvasObject {
	return container.NewVBox(
		c.buildMakeDefaultSection(),
		layout.NewSpacer(),
		c.buildScanBrowsersSection(),
		layout.NewSpacer(),
		c.buildLanguageSection(),
		layout.NewSpacer(),
		c.buildUiSection(),
	)
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

func (c *Configurator) buildScanBrowsersSection() fyne.CanvasObject {
	scanStatusLabel := widget.NewLabel("")
	scanStatusLabel.TextStyle = fyne.TextStyle{Italic: true}
	scanStatusLabel.Hide()

	setupButton := func(button *widget.Button, alreadyScanned bool) {
		if alreadyScanned {
			button.SetText(i18n.T("config.rescan_browsers"))
		} else {
			button.SetText(i18n.T("config.scan_browsers"))
		}
		button.Enable()
	}

	scanBrowsersButton := widget.NewButton(i18n.T("config.scan_now"), func() {})
	scanBrowsersButton.OnTapped = func() {
		scanBrowsersButton.Disable()
		scanStatusLabel.SetText(i18n.T("config.scan_in_progress"))
		scanStatusLabel.Show()

		go func() {
			err := c.settingsService.ScanBrowsers()
			if err != nil {
				scanStatusLabel.SetText(i18n.T("config.scan_failed"))
				scanBrowsersButton.Enable()
				c.logger.Error("Error scanning browsers", "error", err)
			} else {
				scanStatusLabel.SetText(i18n.T("config.scan_success"))
				isConfigured, _ := c.settingsService.IsConfigured()
				setupButton(scanBrowsersButton, isConfigured)
			}
		}()
	}

	isConfigured, _ := c.settingsService.IsConfigured()
	setupButton(scanBrowsersButton, isConfigured)

	return container.NewVBox(
		widget.NewLabel(i18n.T("config.scan_description")),
		scanBrowsersButton,
		scanStatusLabel,
	)
}

func (c *Configurator) buildLanguageSection() fyne.CanvasObject {
	locales := i18n.AvailableLocales()
	autoLabel := i18n.T("config.language_auto")

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

	return container.NewVBox(hideGuideCheck)
}

func (c *Configurator) getRulesTab() fyne.CanvasObject {
	description := widget.NewLabel(i18n.T("config.rules_description"))
	description.Wrapping = fyne.TextWrapWord

	editButton := widget.NewButton(i18n.T("config.rules_edit_button"), func() {
		configPath := c.settingsService.GetConfigFilePath()
		if err := openFileInEditor(configPath); err != nil {
			c.logger.Error("Error opening config file in editor", "error", err)
		}
	})

	return container.NewVBox(
		description,
		layout.NewSpacer(),
		editButton,
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
		layout.NewSpacer(),
		description,
		layout.NewSpacer(),
		details,
		layout.NewSpacer(),
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
