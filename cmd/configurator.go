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

func (c *Configurator) Run() error {
	w := c.fapp.NewWindow(i18n.T("config.window_title"))

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("config.tab_general"), c.getGeneralTab()),
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

func (c *Configurator) getAboutTab() fyne.CanvasObject {
	icon := widget.NewButtonWithIcon(
		"",
		resources.LinkquisitionIcon,
		func() {
			if err := c.browserService.OpenUrlWithDefaultBrowser("https://github.com/Strobotti/linkquisition"); err != nil {
				c.logger.Error("Error opening URL", "error", err)
			}
		},
	)

	return container.NewBorder(
		container.NewBorder(nil, nil, icon, nil, widget.NewLabel(fmt.Sprintf("Linkquisition %s", version))),
		nil,
		nil,
		nil,
		layout.NewSpacer(),
	)
}
