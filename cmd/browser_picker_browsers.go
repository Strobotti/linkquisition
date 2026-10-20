package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/ui"
)

func (picker *BrowserPicker) makeBrowserButton(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) fyne.CanvasObject {
	var icon fyne.Resource

	iconBytes, err := picker.browserService.GetIconForBrowser(browser)
	if err != nil {
		picker.logger.Debug("Failed to load browser icon", "browser", browser.Name, "error", err)
	} else {
		icon = fyne.NewStaticResource("icon.png", iconBytes)
	}

	return widget.NewButtonWithIcon(
		browser.Name,
		icon,
		picker.browserOpenCallback(browser, urlToOpen, remember, rememberMatchType),
	)
}

func (picker *BrowserPicker) makeHorizontalBrowserButton(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) fyne.CanvasObject {
	callback := picker.browserOpenCallback(browser, urlToOpen, remember, rememberMatchType)

	var iconWidget fyne.CanvasObject
	iconBytes, err := picker.browserService.GetIconForBrowser(browser)
	if err != nil {
		picker.logger.Debug("Failed to load browser icon", "browser", browser.Name, "error", err)
		// Show a placeholder with the first letter when no icon is available
		bg := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 40})
		bg.SetMinSize(fyne.NewSize(horizontalButtonIconSize, horizontalButtonIconSize))
		firstRune, _ := utf8.DecodeRuneInString(browser.Name)
		placeholder := canvas.NewText(string(firstRune), color.NRGBA{R: 80, G: 80, B: 80, A: 255})
		placeholder.TextSize = 36
		placeholder.TextStyle = fyne.TextStyle{Bold: true}
		placeholder.Alignment = fyne.TextAlignCenter
		iconWidget = container.NewStack(bg, container.NewCenter(placeholder))
	} else {
		res := fyne.NewStaticResource("icon.png", iconBytes)
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(horizontalButtonIconSize, horizontalButtonIconSize))
		iconWidget = img
	}

	nameLabel := widget.NewLabel(browser.Name)
	nameLabel.Alignment = fyne.TextAlignCenter
	nameLabel.Truncation = fyne.TextTruncateEllipsis

	content := container.NewVBox(
		container.NewCenter(iconWidget),
		nameLabel,
	)

	return ui.NewTappableContainer(content, callback)
}

func (picker *BrowserPicker) browserOpenCallback(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) func() {
	return func() {
		browser.PrivateMode = picker.privateModeEnabled()
		rem, _ := remember.Get()
		picker.logger.Debug("Opening URL with browser", "browser", browser.Name, "remember", rem, "private", browser.PrivateMode)

		settings := picker.settingsService.GetSettings()
		remType, _ := rememberMatchType.Get()

		if rem {
			uto := linkquisition.NewURL(urlToOpen)
			matchValue, matchErr := uto.GetDomain()

			if matchErr == nil && remType == linkquisition.BrowserMatchTypeSite {
				matchValue, matchErr = uto.GetSite()
			}

			if matchErr != nil || matchValue == "" {
				picker.logger.Error(
					"Unable to save browser rule for URL",
					"url", urlToOpen,
					"match_type", remType,
					"error", matchErr,
				)
			} else {
				settings.AddRuleToBrowser(&browser, remType, matchValue)
				if writeErr := picker.settingsService.WriteSettings(settings); writeErr != nil {
					picker.logger.Error("Failed to write settings", "error", writeErr)
				}
			}
		}

		go func() {
			if err := picker.browserService.OpenUrlWithBrowser(urlToOpen, &browser); err != nil {
				picker.logger.Error("Failed to open URL in browser", "url", urlToOpen, "browser", browser.Name, "error", err)
			}

			fyne.Do(picker.fapp.Quit)
		}()
	}
}
