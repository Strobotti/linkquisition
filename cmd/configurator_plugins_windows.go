//go:build windows

package main

import "fyne.io/fyne/v2"

// getPluginsTab returns nil on Windows — plugins are not supported.
// The configurator will skip this tab when building the tab list.
func (c *Configurator) getPluginsTab() fyne.CanvasObject {
	return nil
}
