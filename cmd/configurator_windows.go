//go:build windows

package main

import "fyne.io/fyne/v2"

// buildPlatformNote returns nil on Windows — no platform-specific notes needed.
func (c *Configurator) buildPlatformNote() fyne.CanvasObject {
	return nil
}
