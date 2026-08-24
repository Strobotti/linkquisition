//go:build windows

package main

import "fyne.io/fyne/v2/container"

// pluginTabItems returns no tab items on Windows — plugins are not supported.
func (c *Configurator) pluginTabItems() []*container.TabItem {
	return nil
}
