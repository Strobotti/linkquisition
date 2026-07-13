package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableContainer wraps a canvas object to make it respond to tap and hover events.
// It shows a rounded hover highlight and invokes a callback on tap.
type TappableContainer struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	bg       *canvas.Rectangle
	OnTapped func()
}

// Compile-time interface checks.
var (
	_ fyne.Tappable     = (*TappableContainer)(nil)
	_ desktop.Hoverable = (*TappableContainer)(nil)
)

const tappableCornerRadius = 8

// NewTappableContainer creates a new tappable container with hover highlighting.
func NewTappableContainer(content fyne.CanvasObject, onTapped func()) *TappableContainer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = tappableCornerRadius
	t := &TappableContainer{
		content:  content,
		bg:       bg,
		OnTapped: onTapped,
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *TappableContainer) Tapped(_ *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *TappableContainer) MouseIn(_ *desktop.MouseEvent) {
	t.bg.FillColor = ColorHoverBg
	t.bg.Refresh()
}

func (t *TappableContainer) MouseMoved(_ *desktop.MouseEvent) {}

func (t *TappableContainer) MouseOut() {
	t.bg.FillColor = color.Transparent
	t.bg.Refresh()
}

func (t *TappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(t.bg, t.content))
}
