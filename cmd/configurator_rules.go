package main

import (
	"fmt"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) getRulesTab() fyne.CanvasObject {
	content := container.NewVBox()

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder(i18n.T("config.rules_filter_placeholder"))

	filterEntry.OnChanged = func(text string) {
		c.rebuildRulesList(content, text)
	}

	c.rebuildRulesList(content, "")

	return container.NewBorder(filterEntry, nil, nil, nil, container.NewVScroll(content))
}

func (c *Configurator) rebuildRulesList(content *fyne.Container, filter string) {
	content.RemoveAll()

	settings := c.settingsService.GetSettings()

	if len(settings.Browsers) == 0 {
		emptyLabel := widget.NewLabel(i18n.T("config.rules_no_browsers"))
		emptyLabel.Wrapping = fyne.TextWrapWord
		content.Add(emptyLabel)
		content.Refresh()
		return
	}

	hasAnyRules := false
	filterLower := strings.ToLower(filter)

	for idx := range settings.Browsers {
		b := settings.Browsers[idx]
		if b.Hidden {
			continue
		}

		section := c.buildBrowserRulesSection(settings, idx, content, filterLower)
		if section != nil {
			content.Add(section)
			hasAnyRules = true
		}
	}

	// Also show hidden browsers that have rules
	for idx := range settings.Browsers {
		b := settings.Browsers[idx]
		if !b.Hidden || len(b.Matches) == 0 {
			continue
		}

		section := c.buildBrowserRulesSection(settings, idx, content, filterLower)
		if section != nil {
			content.Add(section)
			hasAnyRules = true
		}
	}

	if !hasAnyRules {
		if filter != "" {
			noMatchLabel := widget.NewLabel(i18n.T("config.rules_no_match"))
			noMatchLabel.TextStyle = fyne.TextStyle{Italic: true}
			content.Add(noMatchLabel)
		} else {
			emptyLabel := widget.NewLabel(i18n.T("config.rules_empty"))
			emptyLabel.Wrapping = fyne.TextWrapWord
			content.Add(emptyLabel)
		}
	}

	content.Refresh()
}

func (c *Configurator) buildBrowserRulesSection(
	settings *linkquisition.Settings, browserIdx int, listContainer *fyne.Container, filter string,
) fyne.CanvasObject {
	b := settings.Browsers[browserIdx]

	// Filter: check if browser name or any rule value matches
	if filter != "" {
		browserNameMatches := strings.Contains(strings.ToLower(b.Name), filter)
		hasMatchingRule := false
		for _, m := range b.Matches {
			if strings.Contains(strings.ToLower(m.Value), filter) {
				hasMatchingRule = true
				break
			}
		}
		if !browserNameMatches && !hasMatchingRule {
			return nil
		}
	}

	// Section header
	title := widget.NewLabel(b.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	addBtn := widget.NewButton(i18n.T("config.rules_add"), func() {
		c.showAddRuleDialog(browserIdx, listContainer)
	})

	header := container.NewBorder(nil, nil, title, addBtn)

	// Rules list
	rulesBox := container.NewVBox()

	if len(b.Matches) == 0 {
		noRules := widget.NewLabel(i18n.T("config.rules_none_for_browser"))
		noRules.TextStyle = fyne.TextStyle{Italic: true}
		rulesBox.Add(noRules)
	} else {
		for ruleIdx := range b.Matches {
			rulesBox.Add(c.buildRuleRow(browserIdx, ruleIdx, listContainer))
		}
	}

	card := container.NewVBox(header, rulesBox)

	return widget.NewCard("", "", card)
}

func (c *Configurator) buildRuleRow(
	browserIdx, ruleIdx int, listContainer *fyne.Container,
) fyne.CanvasObject {
	settings := c.settingsService.GetSettings()
	rule := settings.Browsers[browserIdx].Matches[ruleIdx]

	typeLabel := widget.NewLabel(fmt.Sprintf("[%s]", rule.Type))
	typeLabel.TextStyle = fyne.TextStyle{Bold: true}

	valueLabel := widget.NewLabel(rule.Value)
	valueLabel.Truncation = fyne.TextTruncateEllipsis

	deleteBtn := widget.NewButton("🗑", func() {
		c.deleteRule(browserIdx, ruleIdx, listContainer)
	})

	return container.NewBorder(nil, nil, typeLabel, deleteBtn, valueLabel)
}

func (c *Configurator) showAddRuleDialog(browserIdx int, listContainer *fyne.Container) {
	ruleTypes := []string{
		linkquisition.BrowserMatchTypeSite,
		linkquisition.BrowserMatchTypeDomain,
		linkquisition.BrowserMatchTypeRegex,
	}

	typeSelect := widget.NewSelect(ruleTypes, nil)
	typeSelect.SetSelected(linkquisition.BrowserMatchTypeSite)

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder(i18n.T("config.rules_value_placeholder"))

	errorLabel := widget.NewLabel("")
	errorLabel.TextStyle = fyne.TextStyle{Italic: true}
	errorLabel.Hide()

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(i18n.T("config.rules_type"), typeSelect),
			widget.NewFormItem(i18n.T("config.rules_value"), valueEntry),
		),
		errorLabel,
	)

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	settings := c.settingsService.GetSettings()
	browserName := settings.Browsers[browserIdx].Name

	d := dialog.NewCustomConfirm(
		i18n.T("config.rules_add_title", map[string]interface{}{"Name": browserName}),
		i18n.T("config.plugins_save"),
		i18n.T("config.plugins_cancel"),
		form,
		func(save bool) {
			if !save {
				return
			}
			if valueEntry.Text == "" {
				return
			}

			// Validate regex
			if typeSelect.Selected == linkquisition.BrowserMatchTypeRegex {
				if _, err := regexp.Compile(valueEntry.Text); err != nil {
					errorLabel.SetText(i18n.T("config.rules_regex_invalid"))
					errorLabel.Show()
					return
				}
			}

			c.addRule(browserIdx, typeSelect.Selected, valueEntry.Text, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 220)) //nolint:mnd
	d.Show()
}

func (c *Configurator) addRule(browserIdx int, matchType, matchValue string, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()

	settings.Browsers[browserIdx].Matches = append(
		settings.Browsers[browserIdx].Matches,
		linkquisition.BrowserMatch{
			Type:  matchType,
			Value: matchValue,
		},
	)

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error adding rule", "error", err)
		return
	}

	c.rebuildRulesList(listContainer, "")
}

func (c *Configurator) deleteRule(browserIdx, ruleIdx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	b := &settings.Browsers[browserIdx]

	b.Matches = append(b.Matches[:ruleIdx], b.Matches[ruleIdx+1:]...)

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error deleting rule", "error", err)
		return
	}

	c.rebuildRulesList(listContainer, "")
}
