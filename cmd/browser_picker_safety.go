package main

import (
	"context"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/safety"
	"github.com/strobotti/linkquisition/internal/ui"
)

func (picker *BrowserPicker) buildSafetyIndicator(
	urlToOpen string,
	w fyne.Window,
	settings *linkquisition.Settings,
) fyne.CanvasObject {
	indicator := canvas.NewText("●", ui.ColorNeutral)
	indicator.TextSize = theme.TextSize() * 2

	refButton := widget.NewButton("", nil)
	buttonSize := refButton.MinSize()

	var checkResult *safety.CheckResult
	dotBox := container.New(layout.NewCenterLayout(), indicator)

	indicatorButton := widget.NewButton("", func() {
		if checkResult != nil {
			picker.showSafetyReport(checkResult, w)
		}
	})
	indicatorButton.Importance = widget.LowImportance

	stacked := container.NewStack(indicatorButton, container.NewCenter(dotBox))
	stacked.Resize(fyne.NewSize(buttonSize.Height, buttonSize.Height))

	go func() {
		provider := settings.Security.GetProvider()
		var cache *safety.Cache
		if settings.Security.Cache.Enabled {
			cache = safety.NewCache(
				picker.settingsService.GetConfigFolderPath(),
				provider,
				settings.Security.Cache.GetTTL(),
			)
			if cached, ok := cache.Get(urlToOpen); ok {
				checkResult = cached
				fyne.Do(func() {
					setSafetyIndicatorLevel(indicator, cached.Level)
				})
				return
			}
		}

		checker, err := safety.NewChecker(provider, settings.Security.APIKey)
		if err != nil {
			picker.logger.Error("Failed to create safety checker", "error", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), safetyCheckTimeout)
		defer cancel()

		result, err := checker.Check(ctx, urlToOpen)
		if err != nil {
			picker.logger.Error("Safety check failed", "url", urlToOpen, "error", err)
			fyne.Do(func() {
				indicator.Color = ui.ColorWarning
				indicator.Refresh()
			})
			return
		}

		checkResult = result
		if cache != nil {
			if cacheErr := cache.Put(urlToOpen, result); cacheErr != nil {
				picker.logger.Debug("Failed to cache safety result", "url", urlToOpen, "error", cacheErr)
			}
		}

		fyne.Do(func() {
			setSafetyIndicatorLevel(indicator, result.Level)
		})
	}()

	return stacked
}

func setSafetyIndicatorLevel(indicator *canvas.Text, level safety.ThreatLevel) {
	switch level {
	case safety.ThreatLevelSafe:
		indicator.Color = ui.ColorSuccess
	case safety.ThreatLevelSuspicious:
		indicator.Color = ui.ColorWarning
	case safety.ThreatLevelDangerous:
		indicator.Color = ui.ColorDanger
	}
	indicator.Refresh()
}

func (picker *BrowserPicker) showSafetyReport(result *safety.CheckResult, w fyne.Window) {
	var levelText string
	var levelColor color.Color

	switch result.Level {
	case safety.ThreatLevelSafe:
		levelText = i18n.T("picker.safety_level_safe")
		levelColor = ui.ColorSuccess
	case safety.ThreatLevelSuspicious:
		levelText = i18n.T("picker.safety_level_suspicious")
		levelColor = ui.ColorWarning
	case safety.ThreatLevelDangerous:
		levelText = i18n.T("picker.safety_level_dangerous")
		levelColor = ui.ColorDanger
	}

	levelLabel := canvas.NewText(levelText, levelColor)
	levelLabel.TextStyle = fyne.TextStyle{Bold: true}
	levelLabel.TextSize = theme.TextSize()

	grid := container.New(layout.NewFormLayout(),
		ui.FormLabel(i18n.T("picker.safety_provider")), ui.FormValue(result.Provider),
		ui.FormLabel(i18n.T("picker.safety_result")), levelLabel,
	)
	if len(result.Details) > 0 {
		grid.Add(ui.FormLabel(i18n.T("picker.safety_details")))
		grid.Add(ui.FormValue(strings.Join(result.Details, "\n")))
	}

	var reportLink fyne.CanvasObject
	if result.ReportURL != "" {
		opener := picker.urlOpener()
		reportLink = ui.NewLinkWithCopy(i18n.T("picker.safety_view_report"), result.ReportURL, w, opener)
	}

	closeButton := widget.NewButtonWithIcon(i18n.T("picker.safety_close"), theme.CancelIcon(), nil)
	minWidthSpacer := canvas.NewRectangle(color.Transparent)
	minWidthSpacer.SetMinSize(fyne.NewSize(350, 0)) //nolint:mnd

	content := container.NewVBox(minWidthSpacer, grid)
	if reportLink != nil {
		content.Add(container.NewCenter(reportLink))
	}
	content.Add(container.NewCenter(closeButton))

	popup := widget.NewModalPopUp(content, w.Canvas())
	picker.modalOpen = true
	originalHandler := w.Canvas().OnTypedKey()
	closeModal := func() {
		popup.Hide()
		picker.modalOpen = false
		w.Canvas().SetOnTypedKey(originalHandler)
	}
	closeButton.OnTapped = closeModal
	w.Canvas().SetOnTypedKey(func(keyEvent *fyne.KeyEvent) {
		if keyEvent.Name == fyne.KeyEscape {
			closeModal()
		}
	})
	popup.Show()
}
