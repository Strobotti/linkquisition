package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
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
	w := c.fapp.NewWindow("Linkquisition settings")

	tabs := container.NewAppTabs(
		container.NewTabItem("General", c.getGeneralTab()),
		container.NewTabItem("About", c.getAboutTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(500, 400)) //nolint:gomnd
	w.CenterOnScreen()

	w.ShowAndRun()

	return nil
}

func (c *Configurator) getGeneralTab() fyne.CanvasObject {
	// MAKE DEFAULT -LABEL
	makeDefaultLabel := widget.NewLabel(
		"In order to Linkquisition to function as a browser-picker\n" +
			"it has to be set as the default browser:",
	)

	setupMakeDefaultButton := func(button *widget.Button, isDefault bool) {
		if isDefault {
			button.SetText("All good!")
			button.Disable()
		} else {
			button.SetText("Make default")
			button.Enable()
		}
	}

	// MAKE DEFAULT -BUTTON
	onClickMakeDefaultButton := func(button *widget.Button) {
		button.Disable()
		err := c.browserService.MakeUsTheDefaultBrowser()
		if err != nil {
			button.SetText("Error making default!")
			button.Enable()
			fmt.Printf("error making Linkquisition the default browser: %v", err)
		} else {
			setupMakeDefaultButton(button, true)
		}
	}

	makeDefaultButton := widget.NewButton("checking", func() {})
	makeDefaultButton.OnTapped = func() {
		onClickMakeDefaultButton(makeDefaultButton)
	}
	makeDefaultButton.Disable()

	setupMakeDefaultButton(makeDefaultButton, c.browserService.AreWeTheDefaultBrowser())

	// SCAN BROWSERS -BUTTON
	setupScanBrowsersButton := func(button *widget.Button, alreadyScanned bool) {
		if alreadyScanned {
			button.SetText("Re-scan browsers")
		} else {
			button.SetText("Scan browsers")
		}
		button.Enable()
	}
	onClickScanBrowsersButton := func(button *widget.Button) {
		button.Disable()
		err := c.settingsService.ScanBrowsers()
		if err != nil {
			button.SetText("Error scanning browsers!")
			button.Enable()
			fmt.Printf("error scanning browsers: %v", err)
		} else {
			setupScanBrowsersButton(button, c.settingsService.IsConfigured())
		}
	}

	scanBrowsersButton := widget.NewButton("Scan now", func() {})
	scanBrowsersButton.OnTapped = func() {
		onClickScanBrowsersButton(scanBrowsersButton)
	}

	setupScanBrowsersButton(scanBrowsersButton, c.settingsService.IsConfigured())

	return container.NewVBox(
		makeDefaultLabel,
		makeDefaultButton,
		layout.NewSpacer(),
		widget.NewLabel(
			"The browsers should be scanned and stored in a configuration file for\n"+
				"faster startup and for enabling custom configuration.\n"+
				"\n"+
				"The scan should be safe to execute at any time: only newly detected\n"+
				"browsers are added and the ones no longer present in the system are\n"+
				"removed.\n\nAny existing rules, ordering or customization shouldn't be affected.",
		),
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
