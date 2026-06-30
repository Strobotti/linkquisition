package main

import (
	"fmt"

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
}

func NewConfigurator(
	fapp fyne.App,
	browserService linkquisition.BrowserService,
	settingsService linkquisition.SettingsService,
) *Configurator {
	return &Configurator{
		fapp:            fapp,
		browserService:  browserService,
		settingsService: settingsService,
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
	// MAKE DEFAULT -LABEL
	makeDefaultLabel := widget.NewLabel(i18n.T("config.make_default_label"))

	setupMakeDefaultButton := func(button *widget.Button, isDefault bool) {
		if isDefault {
			button.SetText(i18n.T("config.make_default_done"))
			button.Disable()
		} else {
			button.SetText(i18n.T("config.make_default_button"))
			button.Enable()
		}
	}

	// MAKE DEFAULT -BUTTON
	onClickMakeDefaultButton := func(button *widget.Button) {
		button.Disable()
		err := c.browserService.MakeUsTheDefaultBrowser()
		if err != nil {
			button.SetText(i18n.T("config.make_default_error"))
			button.Enable()
			fmt.Printf("error making Linkquisition the default browser: %v", err)
		} else {
			setupMakeDefaultButton(button, true)
		}
	}

	makeDefaultButton := widget.NewButton(i18n.T("config.make_default_checking"), func() {})
	makeDefaultButton.OnTapped = func() {
		onClickMakeDefaultButton(makeDefaultButton)
	}
	makeDefaultButton.Disable()

	setupMakeDefaultButton(makeDefaultButton, c.browserService.AreWeTheDefaultBrowser())

	// SCAN BROWSERS -BUTTON
	setupScanBrowsersButton := func(button *widget.Button, alreadyScanned bool) {
		if alreadyScanned {
			button.SetText(i18n.T("config.rescan_browsers"))
		} else {
			button.SetText(i18n.T("config.scan_browsers"))
		}
		button.Enable()
	}
	onClickScanBrowsersButton := func(button *widget.Button) {
		button.Disable()
		err := c.settingsService.ScanBrowsers()
		if err != nil {
			button.SetText(i18n.T("config.scan_error"))
			button.Enable()
			fmt.Printf("error scanning browsers: %v", err)
		} else {
			isConfigured, _ := c.settingsService.IsConfigured()
			setupScanBrowsersButton(button, isConfigured)
		}
	}

	scanBrowsersButton := widget.NewButton(i18n.T("config.scan_now"), func() {})
	scanBrowsersButton.OnTapped = func() {
		onClickScanBrowsersButton(scanBrowsersButton)
	}

	// TODO show a spinner while scanning
	// TODO show a message when scanning is done
	// TODO show a message (instead of the button) if configuration is invalid (corrupted file etc)
	isConfigured, _ := c.settingsService.IsConfigured()
	setupScanBrowsersButton(scanBrowsersButton, isConfigured)

	return container.NewVBox(
		makeDefaultLabel,
		makeDefaultButton,
		layout.NewSpacer(),
		widget.NewLabel(i18n.T("config.scan_description")),
		scanBrowsersButton,
	)
}

func (c *Configurator) getAboutTab() fyne.CanvasObject {
	icon := widget.NewButtonWithIcon(
		"",
		resources.LinkquisitionIcon,
		func() {
			if err := c.browserService.OpenUrlWithDefaultBrowser("https://github.com/Strobotti/linkquisition"); err != nil {
				fmt.Printf("error opening url: %s", err.Error())
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
