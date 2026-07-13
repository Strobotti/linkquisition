package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// WithAltRowBackground wraps a widget in a container with a very subtle
// background tint, used for alternating row colors in lists.
func WithAltRowBackground(obj fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(ColorAltRowBg)
	return container.NewStack(bg, obj)
}
