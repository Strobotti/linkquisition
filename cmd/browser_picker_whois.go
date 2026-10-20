package main

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/ui"
	internalwhois "github.com/strobotti/linkquisition/internal/whois"
)

const (
	whoisWindowWidth          = 450
	whoisLoadingHeight        = 100
	whoisErrorHeight          = 150
	whoisContentHeightPadding = 20
)

func (picker *BrowserPicker) showWhoisWindow(urlToOpen string) {
	whoisWindow := picker.fapp.NewWindow(i18n.T("picker.whois_title"))

	whoisWindow.Canvas().SetOnTypedKey(func(keyEvent *fyne.KeyEvent) {
		if keyEvent.Name == fyne.KeyEscape {
			whoisWindow.Close()
		}
	})

	loading := widget.NewProgressBarInfinite()
	loadingLabel := widget.NewLabel(i18n.T("picker.whois_loading"))
	loadingLabel.Alignment = fyne.TextAlignCenter
	whoisWindow.SetContent(container.NewVBox(loadingLabel, loading))
	whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, whoisLoadingHeight))
	whoisWindow.SetFixedSize(true)
	whoisWindow.CenterOnScreen()

	ctx, cancel := context.WithTimeout(context.Background(), whoisTimeout)
	whoisWindow.SetOnClosed(cancel)

	go func() {
		info, err := internalwhois.Lookup(ctx, urlToOpen)
		fyne.Do(func() {
			if err != nil {
				picker.logger.Error("WHOIS lookup failed", "url", urlToOpen, "error", err)
				whoisWindow.SetContent(picker.buildWhoisError(err, whoisWindow))
				whoisWindow.SetFixedSize(false)
				whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, whoisErrorHeight))
				whoisWindow.SetFixedSize(true)
				return
			}

			content := picker.buildWhoisContent(info, whoisWindow)
			whoisWindow.SetContent(content)
			whoisWindow.SetFixedSize(false)
			whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, content.MinSize().Height+whoisContentHeightPadding))
			whoisWindow.SetFixedSize(true)
		})
	}()

	whoisWindow.Show()
}

func (picker *BrowserPicker) buildWhoisContent(
	info *internalwhois.DomainInfo, w fyne.Window,
) fyne.CanvasObject {
	dnssecText := "✗"
	dnssecColor := ui.ColorDanger
	if info.DNSSec {
		dnssecText = "✓"
		dnssecColor = ui.ColorSuccess
	}

	dnssecLabel := canvas.NewText(dnssecText, dnssecColor)
	dnssecLabel.TextStyle = fyne.TextStyle{Bold: true}
	dnssecLabel.TextSize = theme.TextSize()

	grid := container.New(layout.NewFormLayout(),
		ui.FormLabel(i18n.T("picker.whois_domain")), ui.FormValue(info.Domain),
		ui.FormLabel(i18n.T("picker.whois_registrar")), ui.FormValue(info.Registrar),
		ui.FormLabel(i18n.T("picker.whois_created")), ui.FormValue(info.CreatedDate),
		ui.FormLabel(i18n.T("picker.whois_expires")), ui.FormValue(info.ExpiryDate),
		ui.FormLabel(i18n.T("picker.whois_updated")), ui.FormValue(info.UpdatedDate),
		ui.FormLabel(i18n.T("picker.whois_age")), ui.FormValue(info.DomainAge),
		ui.FormLabel(i18n.T("picker.whois_dnssec")), dnssecLabel,
	)

	if len(info.NameServers) > 0 {
		grid.Add(ui.FormLabel(i18n.T("picker.whois_nameservers")))
		grid.Add(ui.FormValue(strings.Join(info.NameServers, ", ")))
	}
	if len(info.Status) > 0 {
		grid.Add(ui.FormLabel(i18n.T("picker.whois_status")))
		grid.Add(ui.FormValue(strings.Join(info.Status, ", ")))
	}

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.whois_close"),
		theme.CancelIcon(),
		func() { w.Close() },
	)

	return container.NewVBox(grid, container.NewCenter(closeButton))
}

func (picker *BrowserPicker) buildWhoisError(err error, w fyne.Window) fyne.CanvasObject {
	errorMsg := i18n.T("picker.whois_error_generic")
	hint := ""

	errText := err.Error()
	switch {
	case strings.Contains(errText, "timed out") || strings.Contains(errText, "deadline exceeded"):
		errorMsg = i18n.T("picker.whois_error_timeout")
		hint = i18n.T("picker.whois_error_hint_network")
	case strings.Contains(errText, "connection refused") || strings.Contains(errText, "no route"):
		errorMsg = i18n.T("picker.whois_error_connection")
		hint = i18n.T("picker.whois_error_hint_network")
	case strings.Contains(errText, "no host") || strings.Contains(errText, "parse"):
		errorMsg = i18n.T("picker.whois_error_invalid_url")
	}

	errLabel := widget.NewLabel(errorMsg)
	errLabel.Wrapping = fyne.TextWrapWord
	errLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(errLabel)
	if hint != "" {
		hintLabel := widget.NewLabel(hint)
		hintLabel.Wrapping = fyne.TextWrapWord
		hintLabel.Alignment = fyne.TextAlignCenter
		hintLabel.Importance = widget.MediumImportance
		content.Add(hintLabel)
	}

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.whois_close"),
		theme.CancelIcon(),
		func() { w.Close() },
	)
	content.Add(container.NewCenter(closeButton))

	return content
}
