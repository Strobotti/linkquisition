package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// FormLabel creates a bold label suitable for form-style layouts (key-value grids).
func FormLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

// FormValue creates a wrapping label for form values, substituting "—" for empty strings.
func FormValue(value string) *widget.Label {
	if value == "" {
		value = "—"
	}
	v := widget.NewLabel(value)
	v.Wrapping = fyne.TextWrapWord
	return v
}
