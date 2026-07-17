package ui

import (
	"log/slog"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const copyFeedbackDuration = 1500 * time.Millisecond

// URLOpener is a function that opens a URL string externally.
// When provided to NewLinkWithCopy, it replaces Fyne's default OpenURL behavior.
// This is necessary when the app is the default browser to avoid circular loops.
type URLOpener func(rawURL string) error

// NewLinkWithCopy creates a hyperlink with a small copy-to-clipboard button beside it.
// The link is displayed using Fyne's Hyperlink widget (underlined, themed color).
// The copy button shows a clipboard icon and briefly switches to a checkmark on tap.
//
// An optional URLOpener can be passed to override the default link-opening behavior.
// This is needed when Linkquisition is the default browser — without it, Fyne's
// built-in OpenURL calls the system handler which loops back to Linkquisition.
func NewLinkWithCopy(text, rawURL string, w fyne.Window, opener ...URLOpener) fyne.CanvasObject {
	parsedURL, _ := url.Parse(rawURL)

	link := widget.NewHyperlink(text, parsedURL)

	// If a custom opener is provided, override the hyperlink's default tap behavior
	if len(opener) > 0 && opener[0] != nil {
		fn := opener[0]
		link.URL = nil // disable default Fyne OpenURL behavior
		link.OnTapped = func() {
			slog.Debug("Link tapped, opening URL with custom opener", "url", rawURL)

			if err := fn(rawURL); err != nil {
				slog.Error("Failed to open URL via custom opener", "url", rawURL, "error", err)
			}
		}
	} else {
		slog.Debug("NewLinkWithCopy created without custom opener, using Fyne default",
			"url", rawURL)
	}

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), nil)
	copyBtn.Importance = widget.LowImportance
	copyBtn.OnTapped = func() {
		w.Clipboard().SetContent(rawURL)

		copyBtn.SetIcon(theme.ConfirmIcon())

		time.AfterFunc(copyFeedbackDuration, func() {
			fyne.Do(func() {
				copyBtn.SetIcon(theme.ContentCopyIcon())
			})
		})
	}

	return container.NewHBox(link, copyBtn)
}
