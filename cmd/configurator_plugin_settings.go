package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

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

	// Build the form
	form := widget.NewForm(formItems...)

	title := i18n.T("config.plugins_settings_title", map[string]interface{}{templateKeyName: meta.Name})

	// Get the parent window — use the fyne app's driver to find it
	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	scrollContent := container.NewVScroll(form)
	scrollContent.SetMinSize(fyne.NewSize(550, 300)) //nolint:mnd

	d := dialog.NewCustomConfirm(
		title,
		i18n.T("config.plugins_save"),
		i18n.T("config.plugins_cancel"),
		scrollContent,
		func(save bool) {
			if !save {
				return
			}
			c.savePluginSettings(pluginIdx, editors, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(600, 450)) //nolint:mnd
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
	current := getSettingStringList(currentSettings, desc.Key)
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

func getSettingStringList(settings map[string]interface{}, key string) []string {
	v, ok := settings[key]
	if !ok {
		return nil
	}

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
