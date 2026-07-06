//go:build darwin

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition/internal/i18n"
)

// buildPlatformNote returns a macOS-specific note informing the user that
// the browser picker is not available while the settings window is open.
func (c *Configurator) buildPlatformNote() fyne.CanvasObject {
	note := widget.NewLabel(i18n.T("config.macos_picker_note"))
	note.Wrapping = fyne.TextWrapWord
	note.TextStyle = fyne.TextStyle{Italic: true}

	return note
}
