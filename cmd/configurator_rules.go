package main

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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
	sectionCount := 0

	for idx := range settings.Browsers {
		b := settings.Browsers[idx]
		if b.Hidden {
			continue
		}

		section := c.buildBrowserRulesSection(settings, idx, content, filterLower)
		if section != nil {
			if sectionCount > 0 {
				content.Add(widget.NewSeparator())
			}
			content.Add(section)
			hasAnyRules = true
			sectionCount++
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
			if sectionCount > 0 {
				content.Add(widget.NewSeparator())
			}
			content.Add(section)
			hasAnyRules = true
			sectionCount++
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

	if !browserMatchesFilter(&b, filter) {
		return nil
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
			row := c.buildRuleRow(browserIdx, ruleIdx, listContainer)
			if ruleIdx%2 == 1 {
				row = withSubtleBackground(row)
			}
			rulesBox.Add(row)
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

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		c.showEditRuleDialog(browserIdx, ruleIdx, listContainer)
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		c.deleteRule(browserIdx, ruleIdx, listContainer)
	})

	buttons := container.NewHBox(editBtn, deleteBtn)

	return container.NewBorder(nil, nil, typeLabel, buttons, valueLabel)
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

	regexIndicator := newRegexIndicator()

	typeSelect.OnChanged = func(selected string) {
		regexIndicator.update(selected, valueEntry.Text)
	}
	valueEntry.OnChanged = func(text string) {
		regexIndicator.update(typeSelect.Selected, text)
	}

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(i18n.T("config.rules_type"), typeSelect),
			widget.NewFormItem(i18n.T("config.rules_value"), container.NewBorder(
				nil, nil, nil, regexIndicator.container, valueEntry,
			)),
		),
	)

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	settings := c.settingsService.GetSettings()
	browserName := settings.Browsers[browserIdx].Name

	d := dialog.NewCustomConfirm(
		i18n.T("config.rules_add_title", map[string]interface{}{templateKeyName: browserName}),
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
					return
				}
			}

			c.addRule(browserIdx, typeSelect.Selected, valueEntry.Text, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 200)) //nolint:mnd
	d.Show()
}

func (c *Configurator) showEditRuleDialog(browserIdx, ruleIdx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	rule := settings.Browsers[browserIdx].Matches[ruleIdx]

	ruleTypes := []string{
		linkquisition.BrowserMatchTypeSite,
		linkquisition.BrowserMatchTypeDomain,
		linkquisition.BrowserMatchTypeRegex,
	}

	typeSelect := widget.NewSelect(ruleTypes, nil)
	typeSelect.SetSelected(rule.Type)

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder(i18n.T("config.rules_value_placeholder"))
	valueEntry.SetText(rule.Value)

	regexIndicator := newRegexIndicator()
	// Initialize indicator state based on current rule
	regexIndicator.update(rule.Type, rule.Value)

	typeSelect.OnChanged = func(selected string) {
		regexIndicator.update(selected, valueEntry.Text)
	}
	valueEntry.OnChanged = func(text string) {
		regexIndicator.update(typeSelect.Selected, text)
	}

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(i18n.T("config.rules_type"), typeSelect),
			widget.NewFormItem(i18n.T("config.rules_value"), container.NewBorder(
				nil, nil, nil, regexIndicator.container, valueEntry,
			)),
		),
	)

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	d := dialog.NewCustomConfirm(
		i18n.T("config.rules_edit_title"),
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
					return
				}
			}

			c.updateRule(browserIdx, ruleIdx, typeSelect.Selected, valueEntry.Text, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 200)) //nolint:mnd
	d.Show()
}

func (c *Configurator) updateRule(
	browserIdx, ruleIdx int, matchType, matchValue string, listContainer *fyne.Container,
) {
	settings := c.settingsService.GetSettings()

	settings.Browsers[browserIdx].Matches[ruleIdx] = linkquisition.BrowserMatch{
		Type:  matchType,
		Value: matchValue,
	}

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error updating rule", "error", err)
		return
	}

	c.rebuildRulesList(listContainer, "")
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

const (
	regexIndicatorSize = 12
)

// regexIndicator shows a colored dot next to the value entry when the rule
// type is "regex". Green = valid regex, red = invalid regex. Hidden for
// non-regex types.
type regexIndicator struct {
	dot       *canvas.Circle
	container *fyne.Container
}

func newRegexIndicator() *regexIndicator {
	dot := canvas.NewCircle(color.NRGBA{R: 0, G: 180, B: 0, A: 255})
	dot.Resize(fyne.NewSize(regexIndicatorSize, regexIndicatorSize))

	dotContainer := container.NewCenter(dot)
	dotContainer.Hide()

	return &regexIndicator{
		dot:       dot,
		container: dotContainer,
	}
}

func (ri *regexIndicator) update(matchType, value string) {
	if matchType != linkquisition.BrowserMatchTypeRegex {
		ri.container.Hide()
		return
	}

	ri.container.Show()

	if value == "" {
		ri.dot.FillColor = color.NRGBA{R: 180, G: 180, B: 0, A: 255} // yellow for empty
		ri.dot.Refresh()
		return
	}

	if _, err := regexp.Compile(value); err != nil {
		ri.dot.FillColor = color.NRGBA{R: 220, G: 0, B: 0, A: 255} // red
	} else {
		ri.dot.FillColor = color.NRGBA{R: 0, G: 180, B: 0, A: 255} // green
	}

	ri.dot.Refresh()
}

// withSubtleBackground wraps a widget in a container with a very subtle
// background tint, used for alternating row colors in rule lists.
func withSubtleBackground(obj fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(color.NRGBA{R: 128, G: 128, B: 128, A: 15})
	return container.NewStack(bg, obj)
}

// browserMatchesFilter returns true if a browser should be shown given the
// current filter text. An empty filter always matches. Otherwise the browser
// matches if its name contains the filter OR any of its rule values do.
func browserMatchesFilter(b *linkquisition.BrowserSettings, filter string) bool {
	if filter == "" {
		return true
	}

	if strings.Contains(strings.ToLower(b.Name), filter) {
		return true
	}

	for _, m := range b.Matches {
		if strings.Contains(strings.ToLower(m.Value), filter) {
			return true
		}
	}

	return false
}
