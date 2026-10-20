//go:build !windows

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/ui"
)

// buildDefaultSettingsFromMetadata constructs a settings map populated with default values
// from the plugin's metadata descriptors.
func buildDefaultSettingsFromMetadata(meta *linkquisition.PluginMetadata) map[string]interface{} {
	defaults := make(map[string]interface{})
	for i := range meta.Settings {
		desc := &meta.Settings[i]
		if desc.Default == nil {
			continue
		}
		switch desc.Type {
		case linkquisition.SettingTypeStringList:
			if list := convertToStringList(desc.Default); list != nil {
				asIface := make([]interface{}, len(list))
				for j, s := range list {
					asIface[j] = s
				}
				defaults[desc.Key] = asIface
			}
		case linkquisition.SettingTypeString, linkquisition.SettingTypeBool,
			linkquisition.SettingTypeInt, linkquisition.SettingTypeDuration,
			linkquisition.SettingTypeChoice, linkquisition.SettingTypeKeyValueList:
			defaults[desc.Key] = desc.Default
		}
	}
	return defaults
}

// buildPluginMetadataHeader creates a header section displaying plugin description,
// author, and homepage URL. Empty optional fields are omitted.
func (c *Configurator) buildPluginMetadataHeader(
	meta *linkquisition.PluginMetadata, parentWindow fyne.Window,
) fyne.CanvasObject {
	items := make([]fyne.CanvasObject, 0, 4) //nolint:mnd

	// Description (always shown if available)
	if meta.Description != "" {
		desc := widget.NewLabel(meta.Description)
		desc.Wrapping = fyne.TextWrapWord
		items = append(items, desc)
	}

	// Author (optional)
	if meta.Author != "" {
		authorLabel := widget.NewRichTextFromMarkdown("**" + i18n.T("config.plugins_author") + ":** " + meta.Author)
		items = append(items, authorLabel)
	}

	// Homepage URL (optional)
	if meta.URL != "" {
		homepageRow := container.NewHBox(
			ui.NewLinkWithCopy(meta.URL, meta.URL, parentWindow, c.urlOpener()),
		)
		items = append(items, homepageRow)
	}

	if len(items) == 0 {
		return nil
	}

	// Add a separator after the metadata section
	items = append(items, widget.NewSeparator())

	return container.NewVBox(items...)
}

func (c *Configurator) showPluginSettings(pluginIdx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	ps := settings.Plugins[pluginIdx]
	meta := c.getPluginMetadata(ps.Path)

	if ps.Settings == nil {
		ps.Settings = make(map[string]interface{})
	}

	// Build form items from metadata setting descriptors
	editors := make(map[string]func() interface{})
	formItems := make([]*widget.FormItem, 0, len(meta.Settings))

	for i := range meta.Settings {
		desc := &meta.Settings[i]
		item, getValue := c.buildSettingWidget(desc, ps.Settings)
		formItems = append(formItems, item)
		editors[desc.Key] = getValue
	}

	// Build the form inside a container we can swap out
	form := widget.NewForm(formItems...)
	formContainer := container.NewStack(form)

	// "Reset to defaults" button
	resetBtn := widget.NewButton(i18n.T("config.plugins_reset_defaults"), func() {
		defaultSettings := buildDefaultSettingsFromMetadata(&meta)
		newEditors := make(map[string]func() interface{})
		newFormItems := make([]*widget.FormItem, 0, len(meta.Settings))
		for i := range meta.Settings {
			desc := &meta.Settings[i]
			item, getValue := c.buildSettingWidget(desc, defaultSettings)
			newFormItems = append(newFormItems, item)
			newEditors[desc.Key] = getValue
		}
		editors = newEditors
		newForm := widget.NewForm(newFormItems...)
		formContainer.RemoveAll()
		formContainer.Add(newForm)
		formContainer.Refresh()
	})

	title := i18n.T("config.plugins_settings_title", map[string]interface{}{templateKeyName: meta.Name})

	parentWindow := c.parentWindow()
	if parentWindow == nil {
		return
	}

	// Build scrollable content with optional metadata header
	var scrollBody fyne.CanvasObject
	if header := c.buildPluginMetadataHeader(&meta, parentWindow); header != nil {
		scrollBody = container.NewVBox(header, formContainer)
	} else {
		scrollBody = formContainer
	}

	scrollContent := container.NewVScroll(scrollBody)
	scrollContent.SetMinSize(fyne.NewSize(780, 420)) //nolint:mnd

	var d dialog.Dialog

	saveBtn := widget.NewButton(i18n.T("config.plugins_save"), func() {
		c.savePluginSettings(pluginIdx, editors, listContainer)
		d.Hide()
	})
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton(i18n.T("config.plugins_cancel"), func() {
		d.Hide()
	})

	buttonRow := container.NewHBox(resetBtn, layout.NewSpacer(), cancelBtn, saveBtn)
	content := container.NewBorder(nil, buttonRow, nil, nil, scrollContent)

	d = dialog.NewCustomWithoutButtons(title, content, parentWindow)
	d.Resize(fyne.NewSize(900, 700)) //nolint:mnd
	d.Show()
}

// showAddPluginSettings shows a settings dialog for a not-yet-configured plugin.
// If the user clicks Save, the plugin is added to the config with the chosen settings.
// If the plugin has no configurable settings, it is added immediately with defaults.
func (c *Configurator) showAddPluginSettings(
	pluginName string, meta *linkquisition.PluginMetadata, listContainer *fyne.Container,
) {
	// If no settings to configure, add immediately with defaults
	if len(meta.Settings) == 0 {
		c.addPlugin(pluginName, listContainer)
		return
	}

	// Pre-populate with default values from metadata descriptors
	defaultSettings := buildDefaultSettingsFromMetadata(meta)
	editors := make(map[string]func() interface{})
	formItems := make([]*widget.FormItem, 0, len(meta.Settings))

	for i := range meta.Settings {
		desc := &meta.Settings[i]
		item, getValue := c.buildSettingWidget(desc, defaultSettings)
		formItems = append(formItems, item)
		editors[desc.Key] = getValue
	}

	form := widget.NewForm(formItems...)
	formContainer := container.NewStack(form)

	// "Reset to defaults" button
	resetBtn := widget.NewButton(i18n.T("config.plugins_reset_defaults"), func() {
		freshDefaults := buildDefaultSettingsFromMetadata(meta)
		newEditors := make(map[string]func() interface{})
		newFormItems := make([]*widget.FormItem, 0, len(meta.Settings))
		for i := range meta.Settings {
			desc := &meta.Settings[i]
			item, getValue := c.buildSettingWidget(desc, freshDefaults)
			newFormItems = append(newFormItems, item)
			newEditors[desc.Key] = getValue
		}
		editors = newEditors
		newForm := widget.NewForm(newFormItems...)
		formContainer.RemoveAll()
		formContainer.Add(newForm)
		formContainer.Refresh()
	})

	title := i18n.T("config.plugins_settings_title", map[string]interface{}{templateKeyName: meta.Name})

	parentWindow := c.parentWindow()
	if parentWindow == nil {
		return
	}

	// Build scrollable content with optional metadata header
	var scrollBody fyne.CanvasObject
	if header := c.buildPluginMetadataHeader(meta, parentWindow); header != nil {
		scrollBody = container.NewVBox(header, formContainer)
	} else {
		scrollBody = formContainer
	}

	scrollContent := container.NewVScroll(scrollBody)
	scrollContent.SetMinSize(fyne.NewSize(780, 420)) //nolint:mnd

	var d dialog.Dialog

	saveBtn := widget.NewButton(i18n.T("config.plugins_save"), func() {
		// Collect settings from editors
		pluginSettings := make(map[string]interface{})
		for key, getValue := range editors {
			if val := getValue(); val != nil {
				pluginSettings[key] = val
			}
		}
		c.addPluginWithSettings(pluginName, pluginSettings, listContainer)
		d.Hide()
	})
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton(i18n.T("config.plugins_cancel"), func() {
		d.Hide()
	})

	buttonRow := container.NewHBox(resetBtn, layout.NewSpacer(), cancelBtn, saveBtn)
	content := container.NewBorder(nil, buttonRow, nil, nil, scrollContent)

	d = dialog.NewCustomWithoutButtons(title, content, parentWindow)
	d.Resize(fyne.NewSize(900, 700)) //nolint:mnd
	d.Show()
}

func (c *Configurator) savePluginSettings(
	pluginIdx int, editors map[string]func() interface{}, listContainer *fyne.Container,
) {
	settings := c.settingsService.GetSettings()
	if settings.Plugins[pluginIdx].Settings == nil {
		settings.Plugins[pluginIdx].Settings = make(map[string]interface{})
	}

	for key, getValue := range editors {
		val := getValue()
		if val == nil {
			delete(settings.Plugins[pluginIdx].Settings, key)
		} else {
			settings.Plugins[pluginIdx].Settings[key] = val
		}
	}

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error saving plugin settings", "error", err)
	}

	c.rebuildPluginsList(listContainer)
}
