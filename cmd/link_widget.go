package main

import (
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const copyFeedbackDuration = 1500 * time.Millisecond

// newLinkWithCopy creates a hyperlink with a small copy-to-clipboard button beside it.
// The link is displayed using Fyne's Hyperlink widget (underlined, themed color).
// The copy button shows a clipboard icon and briefly switches to a checkmark on tap.
func newLinkWithCopy(text, rawURL string, w fyne.Window) fyne.CanvasObject {
	parsedURL, _ := url.Parse(rawURL)

	link := widget.NewHyperlink(text, parsedURL)

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), nil)
	copyBtn.Importance = widget.LowImportance
	copyBtn.OnTapped = func() {
		w.Clipboard().SetContent(rawURL)

		copyBtn.SetIcon(theme.ConfirmIcon())
		copyBtn.SetText("✓")

		time.AfterFunc(copyFeedbackDuration, func() {
			fyne.Do(func() {
				copyBtn.SetText("")
				copyBtn.SetIcon(theme.ContentCopyIcon())
			})
		})
	}

	return container.NewHBox(link, copyBtn)
}
