package main

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) getPluginsTab() fyne.CanvasObject {
	content := container.NewVBox()
	c.rebuildPluginsList(content)
	return container.NewVScroll(content)
}

func (c *Configurator) rebuildPluginsList(content *fyne.Container) {
	content.RemoveAll()

	settings := c.settingsService.GetSettings()

	// Configured plugins
	for idx := range settings.Plugins {
		if idx > 0 {
			content.Add(widget.NewSeparator())
		}
		content.Add(c.buildPluginCard(settings, idx, content))
	}

	// Reorder note
	if len(settings.Plugins) > 0 {
		note := widget.NewLabel(i18n.T("config.plugins_restart_note"))
		note.TextStyle = fyne.TextStyle{Italic: true}
		content.Add(note)
	}

	// Available but unconfigured plugins
	available := c.discoverUnconfiguredPlugins(settings)
	if len(available) > 0 {
		content.Add(layout.NewSpacer())
		availLabel := widget.NewLabel(i18n.T("config.plugins_available"))
		availLabel.TextStyle = fyne.TextStyle{Bold: true}
		content.Add(availLabel)

		for _, name := range available {
			content.Add(c.buildAvailablePluginRow(name, content))
		}
	} else if len(settings.Plugins) == 0 {
		content.Add(widget.NewLabel(i18n.T("config.plugins_none_available")))
	}

	content.Refresh()
}

func (c *Configurator) buildPluginCard(
	settings *linkquisition.Settings, idx int, listContainer *fyne.Container,
) fyne.CanvasObject {
	ps := settings.Plugins[idx]
	pluginName := pluginDisplayName(ps.Path)

	// Try to get metadata from the .so file
	meta := c.getPluginMetadata(ps.Path)

	// Title and description
	title := widget.NewLabel(meta.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	desc := widget.NewLabel(meta.Description)
	desc.Wrapping = fyne.TextWrapWord

	// Enable/disable toggle
	enableCheck := widget.NewCheck(i18n.T("config.plugins_enabled"), func(checked bool) {
		s := c.settingsService.GetSettings()
		s.Plugins[idx].IsDisabled = !checked
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving plugin state", "error", err, "plugin", pluginName)
		}
	})
	enableCheck.Checked = !ps.IsDisabled

	// Configure button — gear icon, compact
	configBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		c.showPluginSettings(idx, listContainer)
	})

	// Only show configure button if plugin has settings
	if len(meta.Settings) == 0 {
		configBtn.Hide()
	}

	// Delete button — trash icon
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		c.confirmRemovePlugin(idx, meta.Name, listContainer)
	})

	// Reorder buttons
	upBtn := widget.NewButton(i18n.T("config.plugins_move_up"), func() {
		c.movePlugin(idx, -1, listContainer)
	})
	if idx == 0 {
		upBtn.Disable()
	}

	downBtn := widget.NewButton(i18n.T("config.plugins_move_down"), func() {
		c.movePlugin(idx, 1, listContainer)
	})
	if idx == len(settings.Plugins)-1 {
		downBtn.Disable()
	}

	// Layout
	headerRow := container.NewBorder(
		nil, nil,
		container.NewHBox(title),
		container.NewHBox(configBtn, upBtn, downBtn, enableCheck, deleteBtn),
	)

	card := container.NewVBox(headerRow, desc)

	return widget.NewCard("", "", card)
}

func (c *Configurator) buildAvailablePluginRow(name string, listContainer *fyne.Container) fyne.CanvasObject {
	pluginName := name
	meta := c.getPluginMetadata(pluginName + pluginExtension)

	title := widget.NewLabel(meta.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	desc := widget.NewLabel(meta.Description)
	desc.Wrapping = fyne.TextWrapWord

	addBtn := widget.NewButton(i18n.T("config.plugins_add"), func() {
		c.showAddPluginSettings(pluginName, &meta, listContainer)
	})

	headerRow := container.NewBorder(nil, nil, title, addBtn)
	card := container.NewVBox(headerRow, desc)

	return widget.NewCard("", "", card)
}

func (c *Configurator) getPluginMetadata(pluginPath string) linkquisition.PluginMetadata {
	path := c.resolvePluginPath(pluginPath)
	if path == "" {
		return linkquisition.PluginMetadata{
			Name:        pluginDisplayName(pluginPath),
			Description: "(plugin file not found)",
		}
	}

	meta, err := probePluginMetadata(path, c.logger)
	if err != nil {
		c.logger.Warn("Failed to probe plugin metadata", "plugin", pluginPath, "error", err)
		return linkquisition.PluginMetadata{
			Name:        pluginDisplayName(pluginPath),
			Description: "(unable to read plugin metadata)",
		}
	}

	return meta
}

func (c *Configurator) resolvePluginPath(pluginPath string) string {
	return resolvePluginPathFromDisk(pluginPath, c.settingsService.GetPluginFolderPath())
}

// resolvePluginPathFromDisk resolves a plugin path to an absolute path on disk.
// It checks the path as-is first, then tries the plugin folder. Returns "" if not found.
func resolvePluginPathFromDisk(pluginPath, pluginFolder string) string {
	if !strings.HasSuffix(pluginPath, pluginExtension) {
		pluginPath += pluginExtension
	}

	if _, err := os.Stat(pluginPath); err == nil {
		return pluginPath
	}

	resolved := filepath.Join(pluginFolder, pluginPath)
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}

	return ""
}

func (c *Configurator) movePlugin(idx, direction int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	newIdx := idx + direction

	if newIdx < 0 || newIdx >= len(settings.Plugins) {
		return
	}

	settings.Plugins[idx], settings.Plugins[newIdx] = settings.Plugins[newIdx], settings.Plugins[idx]

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error saving plugin order", "error", err)
		return
	}

	c.rebuildPluginsList(listContainer)
}

func (c *Configurator) confirmRemovePlugin(idx int, displayName string, listContainer *fyne.Container) {
	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	dialog.ShowConfirm(
		i18n.T("config.plugins_remove_title"),
		i18n.T("config.plugins_remove_confirm", map[string]interface{}{templateKeyName: displayName}),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			settings := c.settingsService.GetSettings()
			if idx < 0 || idx >= len(settings.Plugins) {
				return
			}
			settings.Plugins = append(settings.Plugins[:idx], settings.Plugins[idx+1:]...)
			if err := c.settingsService.WriteSettings(settings); err != nil {
				c.logger.Error("Error removing plugin", "error", err)
				return
			}
			c.rebuildPluginsList(listContainer)
		},
		parentWindow,
	)
}

func (c *Configurator) addPlugin(name string, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()

	settings.Plugins = append(settings.Plugins, linkquisition.PluginSettings{
		Path:       name + pluginExtension,
		IsDisabled: false,
	})

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error adding plugin", "error", err, "plugin", name)
		return
	}

	c.rebuildPluginsList(listContainer)
}

func (c *Configurator) addPluginWithSettings(
	name string, pluginSettings map[string]interface{}, listContainer *fyne.Container,
) {
	settings := c.settingsService.GetSettings()

	settings.Plugins = append(settings.Plugins, linkquisition.PluginSettings{
		Path:       name + pluginExtension,
		IsDisabled: false,
		Settings:   pluginSettings,
	})

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error adding plugin", "error", err, "plugin", name)
		return
	}

	c.rebuildPluginsList(listContainer)
}

func (c *Configurator) discoverUnconfiguredPlugins(settings *linkquisition.Settings) []string {
	pluginDir := c.settingsService.GetPluginFolderPath()

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil
	}

	configured := make(map[string]bool, len(settings.Plugins))
	for _, p := range settings.Plugins {
		configured[strings.ToLower(pluginDisplayName(p.Path))] = true
	}

	var available []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), pluginExtension) {
			continue
		}
		baseName := strings.TrimSuffix(entry.Name(), pluginExtension)
		if !configured[strings.ToLower(baseName)] {
			available = append(available, baseName)
		}
	}

	return available
}
