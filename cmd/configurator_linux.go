//go:build linux

package main

import "fyne.io/fyne/v2"

// buildPlatformNote returns nil on Linux — no platform-specific notes needed.
func (c *Configurator) buildPlatformNote() fyne.CanvasObject {
	return nil
}
