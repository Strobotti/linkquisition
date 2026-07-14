//go:build !windows

package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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
			ui.NewLinkWithCopy(meta.URL, meta.URL, parentWindow),
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

func (c *Configurator) buildSettingWidget(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	switch desc.Type {
	case linkquisition.SettingTypeBool:
		return c.buildBoolSetting(desc, currentSettings)
	case linkquisition.SettingTypeChoice:
		return c.buildChoiceSetting(desc, currentSettings)
	case linkquisition.SettingTypeInt:
		return c.buildIntSetting(desc, currentSettings)
	case linkquisition.SettingTypeString, linkquisition.SettingTypeDuration:
		return c.buildStringSetting(desc, currentSettings)
	case linkquisition.SettingTypeStringList:
		return c.buildStringListSetting(desc, currentSettings)
	case linkquisition.SettingTypeKeyValueList:
		return c.buildKeyValueListSetting(desc, currentSettings)
	default:
		return c.buildStringSetting(desc, currentSettings)
	}
}

func (c *Configurator) buildBoolSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingBool(currentSettings, desc.Key, desc.Default)
	check := widget.NewCheck("", nil)
	check.Checked = current

	item = widget.NewFormItem(desc.Label, check)
	item.HintText = desc.Description

	return item, func() interface{} { return check.Checked }
}

func (c *Configurator) buildChoiceSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingString(currentSettings, desc.Key, desc.Default)
	sel := widget.NewSelect(desc.Options, nil)
	sel.Selected = current

	item = widget.NewFormItem(desc.Label, sel)
	item.HintText = desc.Description

	return item, func() interface{} {
		if sel.Selected == "" {
			return nil
		}
		return sel.Selected
	}
}

func (c *Configurator) buildIntSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingString(currentSettings, desc.Key, desc.Default)
	entry := widget.NewEntry()
	entry.SetText(current)
	entry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		if _, err := strconv.Atoi(s); err != nil {
			return fmt.Errorf("must be a number")
		}
		return nil
	}

	item = widget.NewFormItem(desc.Label, entry)
	item.HintText = desc.Description

	return item, func() interface{} {
		text := entry.Text
		if text == "" {
			return nil
		}
		if v, err := strconv.Atoi(text); err == nil {
			return v
		}
		return nil
	}
}

func (c *Configurator) buildStringSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingString(currentSettings, desc.Key, desc.Default)
	entry := widget.NewEntry()
	entry.SetText(current)

	if desc.Type == linkquisition.SettingTypeDuration {
		entry.SetPlaceHolder("e.g. 5s, 168h")
	}

	item = widget.NewFormItem(desc.Label, entry)
	item.HintText = desc.Description

	return item, func() interface{} {
		if entry.Text == "" {
			return nil
		}
		return entry.Text
	}
}

func (c *Configurator) buildStringListSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingStringList(currentSettings, desc.Key, desc.Default)
	entry := widget.NewMultiLineEntry()
	entry.SetText(strings.Join(current, "\n"))
	entry.SetMinRowsVisible(3) //nolint:mnd
	entry.SetPlaceHolder("One entry per line")

	item = widget.NewFormItem(desc.Label, entry)
	item.HintText = desc.Description

	return item, func() interface{} {
		text := strings.TrimSpace(entry.Text)
		if text == "" {
			return nil
		}
		lines := strings.Split(text, "\n")
		result := make([]interface{}, 0, len(lines))
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
}

func (c *Configurator) buildKeyValueListSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	keyLabel := desc.KeyFieldLabel
	if keyLabel == "" {
		keyLabel = desc.KeyField
	}
	valueLabel := desc.ValueFieldLabel
	if valueLabel == "" {
		valueLabel = desc.ValueField
	}

	current := getSettingKeyValueListStructured(currentSettings, desc.Key, desc.KeyField, desc.ValueField)

	// rows holds the mutable state of key-value entries
	type rowEntry struct {
		keyEntry   *widget.Entry
		valueEntry *widget.Entry
	}
	rows := make([]*rowEntry, 0, len(current))

	// listContainer holds the rendered rows
	listContainer := container.NewVBox()

	var rebuildList func()
	rebuildList = func() {
		listContainer.RemoveAll()

		// Header row
		headerKey := widget.NewLabel(keyLabel)
		headerKey.TextStyle.Bold = true
		headerVal := widget.NewLabel(valueLabel)
		headerVal.TextStyle.Bold = true
		headerRow := container.NewBorder(nil, nil, nil, widget.NewLabel(""),
			container.NewGridWithColumns(2, headerKey, headerVal), //nolint:mnd
		)
		listContainer.Add(headerRow)

		for i := range rows {
			idx := i
			removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				rows = append(rows[:idx], rows[idx+1:]...)
				rebuildList()
			})
			entries := container.NewGridWithColumns(2, rows[idx].keyEntry, rows[idx].valueEntry) //nolint:mnd
			row := container.NewBorder(nil, nil, nil, removeBtn, entries)
			listContainer.Add(row)
		}
	}

	// Initialize rows from current values
	for _, pair := range current {
		keyE := widget.NewEntry()
		keyE.SetText(pair.Key)
		keyE.SetPlaceHolder(keyLabel)
		valE := widget.NewEntry()
		valE.SetText(pair.Value)
		valE.SetPlaceHolder(valueLabel)
		rows = append(rows, &rowEntry{keyEntry: keyE, valueEntry: valE})
	}

	rebuildList()

	addBtn := widget.NewButton(i18n.T("config.plugins_kv_add_row"), func() {
		keyE := widget.NewEntry()
		keyE.SetPlaceHolder(keyLabel)
		valE := widget.NewEntry()
		valE.SetPlaceHolder(valueLabel)
		rows = append(rows, &rowEntry{keyEntry: keyE, valueEntry: valE})
		rebuildList()
	})

	content := container.NewVBox(listContainer, addBtn)
	item = widget.NewFormItem(desc.Label, content)
	item.HintText = desc.Description

	return item, func() interface{} {
		result := make([]interface{}, 0, len(rows))
		for _, row := range rows {
			k := strings.TrimSpace(row.keyEntry.Text)
			v := strings.TrimSpace(row.valueEntry.Text)
			if k == "" && v == "" {
				continue
			}
			result = append(result, map[string]interface{}{
				desc.KeyField:   k,
				desc.ValueField: v,
			})
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
}

// Helper functions to extract current setting values from the untyped map

func getSettingString(settings map[string]interface{}, key string, defaultVal interface{}) string {
	if v, ok := settings[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	if defaultVal != nil {
		return fmt.Sprintf("%v", defaultVal)
	}
	return ""
}

func getSettingBool(settings map[string]interface{}, key string, defaultVal interface{}) bool {
	if v, ok := settings[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return b == "true"
		}
	}
	if defaultVal != nil {
		if b, ok := defaultVal.(bool); ok {
			return b
		}
	}
	return false
}

func getSettingStringList(settings map[string]interface{}, key string, defaultVal interface{}) []string {
	v, ok := settings[key]
	if !ok {
		if defaultVal != nil {
			return convertToStringList(defaultVal)
		}
		return nil
	}

	return convertToStringList(v)
}

func convertToStringList(v interface{}) []string {
	switch list := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(list))
		for _, item := range list {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case []string:
		return list
	}

	return nil
}

// kvPair represents a single key-value pair for the KeyValueList setting type.
type kvPair struct {
	Key   string
	Value string
}

// getSettingKeyValueListStructured extracts key-value pairs using the actual JSON field names
// from the stored map objects.
func getSettingKeyValueListStructured(
	settings map[string]interface{}, key, keyFieldName, valueFieldName string,
) []kvPair {
	v, ok := settings[key]
	if !ok {
		return nil
	}

	list, ok := v.([]interface{})
	if !ok {
		return nil
	}

	result := make([]kvPair, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		pair := kvPair{
			Key:   fmt.Sprintf("%v", m[keyFieldName]),
			Value: fmt.Sprintf("%v", m[valueFieldName]),
		}
		result = append(result, pair)
	}
	return result
}
