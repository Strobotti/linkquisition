//go:build !windows

package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) buildSettingWidget(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	switch desc.Type {
	case linkquisition.SettingTypeBool:
		return buildBoolSetting(desc, currentSettings)
	case linkquisition.SettingTypeChoice:
		return buildChoiceSetting(desc, currentSettings)
	case linkquisition.SettingTypeInt:
		return buildIntSetting(desc, currentSettings)
	case linkquisition.SettingTypeString, linkquisition.SettingTypeDuration:
		return buildStringSetting(desc, currentSettings)
	case linkquisition.SettingTypeStringList:
		return buildStringListSetting(desc, currentSettings)
	case linkquisition.SettingTypeKeyValueList:
		return buildKeyValueListSetting(desc, currentSettings)
	default:
		return buildStringSetting(desc, currentSettings)
	}
}

func buildBoolSetting(
	desc *linkquisition.PluginSettingDescriptor, currentSettings map[string]interface{},
) (item *widget.FormItem, getValue func() interface{}) {
	current := getSettingBool(currentSettings, desc.Key, desc.Default)
	check := widget.NewCheck("", nil)
	check.Checked = current

	item = widget.NewFormItem(desc.Label, check)
	item.HintText = desc.Description

	return item, func() interface{} { return check.Checked }
}

func buildChoiceSetting(
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

func buildIntSetting(
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

func buildStringSetting(
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

func buildStringListSetting(
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

func buildKeyValueListSetting(
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

	type rowEntry struct {
		keyEntry   *widget.Entry
		valueEntry *widget.Entry
	}
	rows := make([]*rowEntry, 0, len(current))
	listContainer := container.NewVBox()

	var rebuildList func()
	rebuildList = func() {
		listContainer.RemoveAll()

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
			listContainer.Add(container.NewBorder(nil, nil, nil, removeBtn, entries))
		}
	}

	for _, pair := range current {
		keyEntry := widget.NewEntry()
		keyEntry.SetText(pair.Key)
		keyEntry.SetPlaceHolder(keyLabel)
		valueEntry := widget.NewEntry()
		valueEntry.SetText(pair.Value)
		valueEntry.SetPlaceHolder(valueLabel)
		rows = append(rows, &rowEntry{keyEntry: keyEntry, valueEntry: valueEntry})
	}
	rebuildList()

	addButton := widget.NewButton(i18n.T("config.plugins_kv_add_row"), func() {
		keyEntry := widget.NewEntry()
		keyEntry.SetPlaceHolder(keyLabel)
		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder(valueLabel)
		rows = append(rows, &rowEntry{keyEntry: keyEntry, valueEntry: valueEntry})
		rebuildList()
	})

	item = widget.NewFormItem(desc.Label, container.NewVBox(listContainer, addButton))
	item.HintText = desc.Description

	return item, func() interface{} {
		result := make([]interface{}, 0, len(rows))
		for _, row := range rows {
			key := strings.TrimSpace(row.keyEntry.Text)
			value := strings.TrimSpace(row.valueEntry.Text)
			if key == "" && value == "" {
				continue
			}
			result = append(result, map[string]interface{}{
				desc.KeyField:   key,
				desc.ValueField: value,
			})
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
}
