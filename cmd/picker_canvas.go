package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/strobotti/linkquisition"
)

// Compile-time check that fynePickerCanvas implements PickerCanvas.
var _ linkquisition.PickerCanvas = (*fynePickerCanvas)(nil)

// fynePickerCanvas adapts a fyne.Window to the PickerCanvas interface,
// allowing plugins to draw effects without importing fyne directly.
// Effects are rendered as background layers beneath the main content,
// so they do not intercept mouse/touch input.
type fynePickerCanvas struct {
	window  fyne.Window
	rasters []*canvas.Raster
}

func newFynePickerCanvas(window fyne.Window) *fynePickerCanvas {
	return &fynePickerCanvas{window: window}
}

func (c *fynePickerCanvas) AddRasterOverlay(translucency float64, draw func(w, h int) []uint8) {
	raster := canvas.NewRaster(func(w, h int) image.Image {
		if draw == nil || w == 0 || h == 0 {
			return image.NewRGBA(image.Rect(0, 0, 1, 1))
		}

		pixels := draw(w, h)
		img := image.NewRGBA(image.Rect(0, 0, w, h))

		expectedLen := w * h * 4
		if len(pixels) >= expectedLen {
			copy(img.Pix, pixels[:expectedLen])
		} else {
			copy(img.Pix, pixels)
		}

		return img
	})
	raster.Translucency = translucency
	c.rasters = append(c.rasters, raster)
}

func (c *fynePickerCanvas) ScheduleRefresh() {
	fyne.Do(func() {
		for _, r := range c.rasters {
			canvas.Refresh(r)
		}
	})
}

func (c *fynePickerCanvas) Width() int {
	return int(c.window.Canvas().Size().Width)
}

func (c *fynePickerCanvas) Height() int {
	return int(c.window.Canvas().Size().Height)
}

// buildPickerContent wraps the main content with any plugin effect layers underneath.
// Effects are rendered as background layers in a Stack container, so they don't block input.
// The content (buttons, labels) sits on top and receives all mouse/keyboard events normally.
func buildPickerContent(
	content fyne.CanvasObject, window fyne.Window, hooks []linkquisition.PluginUIHook,
) fyne.CanvasObject {
	if len(hooks) == 0 {
		return content
	}

	var layers []fyne.CanvasObject

	// Let each hook add its raster layers
	for _, hook := range hooks {
		pc := newFynePickerCanvas(window)
		hook.OnPickerShown(pc)

		for _, r := range pc.rasters {
			layers = append(layers, r)
		}
	}

	if len(layers) == 0 {
		return content
	}

	// Stack: effects at the bottom, content on top
	layers = append(layers, content)

	return container.NewStack(layers...)
}
