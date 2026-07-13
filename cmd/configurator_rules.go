package main

import (
	"fmt"
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
	"github.com/strobotti/linkquisition/internal/ui"
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

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	regexPanel := newRegexPanel(parentWindow)
	regexPanel.update(linkquisition.BrowserMatchTypeSite, "")

	typeSelect.OnChanged = func(selected string) {
		regexPanel.update(selected, valueEntry.Text)
	}
	valueEntry.OnChanged = func(text string) {
		regexPanel.update(typeSelect.Selected, text)
	}

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(i18n.T("config.rules_type"), typeSelect),
			widget.NewFormItem(i18n.T("config.rules_value"), container.NewBorder(
				nil, nil, nil, regexPanel.indicatorBox, valueEntry,
			)),
		),
		regexPanel.panel,
	)

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
	d.Resize(fyne.NewSize(500, 300)) //nolint:mnd
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

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	regexPanel := newRegexPanel(parentWindow)
	regexPanel.update(rule.Type, rule.Value)

	typeSelect.OnChanged = func(selected string) {
		regexPanel.update(selected, valueEntry.Text)
	}
	valueEntry.OnChanged = func(text string) {
		regexPanel.update(typeSelect.Selected, text)
	}

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem(i18n.T("config.rules_type"), typeSelect),
			widget.NewFormItem(i18n.T("config.rules_value"), container.NewBorder(
				nil, nil, nil, regexPanel.indicatorBox, valueEntry,
			)),
		),
		regexPanel.panel,
	)

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
	d.Resize(fyne.NewSize(500, 300)) //nolint:mnd
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

// regexPanel provides live regex validation and test-matching UI.
// Hidden for non-regex rule types. Shows:
// 1. ✓/✗ indicator next to the value entry
// 2. Error message below value when invalid
// 3. A test URL input with match result indicator
type regexPanel struct {
	indicator     *canvas.Text
	indicatorBox  *fyne.Container
	errorLabel    *widget.Label
	typeHelpLabel *widget.Label
	regexHelpRow  *fyne.Container
	testEntry     *widget.Entry
	testResult    *canvas.Text
	testResultBox *fyne.Container
	panel         *fyne.Container
	pattern       string
}

func newRegexPanel(w fyne.Window) *regexPanel {
	// Indicator next to value entry
	indicator := canvas.NewText("✓", ui.ColorSuccess)
	indicator.TextSize = 18
	indicator.TextStyle = fyne.TextStyle{Bold: true}
	indicatorBox := container.NewCenter(indicator)
	indicatorBox.Hide()

	// Error label shown below value when regex is invalid
	errorLabel := widget.NewLabel("")
	errorLabel.TextStyle = fyne.TextStyle{Italic: true}
	errorLabel.Hide()

	// Test URL entry
	testEntry := widget.NewEntry()
	testEntry.SetPlaceHolder(i18n.T("config.rules_regex_test_placeholder"))

	// Test match result
	testResult := canvas.NewText("", ui.ColorSuccess)
	testResult.TextSize = 14
	testResult.TextStyle = fyne.TextStyle{Bold: true}
	testResultBox := container.NewHBox(testResult)
	testResultBox.Hide()

	// Help text with link to regex reference (only shown for regex type)
	regexHelpLabel := widget.NewLabel(i18n.T("config.rules_regex_help"))
	regexHelpLabel.TextStyle = fyne.TextStyle{Italic: true}
	regexDocsURL := "https://pkg.go.dev/regexp/syntax"
	regexHelpLink := ui.NewLinkWithCopy(i18n.T("config.rules_regex_help_link"), regexDocsURL, w)
	regexHelpRow := container.NewHBox(regexHelpLabel, regexHelpLink)
	regexHelpRow.Hide()

	// Type help label (shown for all types, content changes with type)
	typeHelpLabel := widget.NewLabel("")
	typeHelpLabel.Wrapping = fyne.TextWrapWord
	typeHelpLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Full panel (everything below the value entry)
	panel := container.NewVBox(errorLabel, typeHelpLabel, regexHelpRow, testEntry, testResultBox)
	panel.Hide()

	rp := &regexPanel{
		indicator:     indicator,
		indicatorBox:  indicatorBox,
		errorLabel:    errorLabel,
		typeHelpLabel: typeHelpLabel,
		regexHelpRow:  regexHelpRow,
		testEntry:     testEntry,
		testResult:    testResult,
		testResultBox: testResultBox,
		panel:         panel,
	}

	testEntry.OnChanged = func(_ string) {
		rp.refreshTestResult()
	}

	return rp
}

func (rp *regexPanel) update(matchType, value string) {
	rp.panel.Show()
	rp.pattern = value

	// Update type-specific help text
	switch matchType {
	case linkquisition.BrowserMatchTypeSite:
		rp.typeHelpLabel.SetText(i18n.T("config.rules_type_help_site"))
		rp.typeHelpLabel.Show()
	case linkquisition.BrowserMatchTypeDomain:
		rp.typeHelpLabel.SetText(i18n.T("config.rules_type_help_domain"))
		rp.typeHelpLabel.Show()
	case linkquisition.BrowserMatchTypeRegex:
		rp.typeHelpLabel.SetText(i18n.T("config.rules_type_help_regex"))
		rp.typeHelpLabel.Show()
	default:
		rp.typeHelpLabel.Hide()
	}

	// Regex-specific elements
	if matchType != linkquisition.BrowserMatchTypeRegex {
		rp.indicatorBox.Hide()
		rp.regexHelpRow.Hide()
		rp.testEntry.Hide()
		rp.testResultBox.Hide()
		rp.errorLabel.Hide()
		return
	}

	rp.indicatorBox.Show()
	rp.regexHelpRow.Show()
	rp.testEntry.Show()

	if value == "" {
		rp.indicator.Text = "—"
		rp.indicator.Color = ui.ColorNeutral
		rp.indicator.Refresh()
		rp.errorLabel.Hide()
		rp.refreshTestResult()
		return
	}

	if _, err := regexp.Compile(value); err != nil {
		rp.indicator.Text = "✗"
		rp.indicator.Color = ui.ColorDanger
		rp.errorLabel.SetText("✗ " + i18n.T("config.rules_regex_invalid"))
		rp.errorLabel.Show()
	} else {
		rp.indicator.Text = "✓"
		rp.indicator.Color = ui.ColorSuccess
		rp.errorLabel.Hide()
	}

	rp.indicator.Refresh()
	rp.refreshTestResult()
}

func (rp *regexPanel) refreshTestResult() {
	testURL := rp.testEntry.Text
	if testURL == "" {
		rp.testResultBox.Hide()
		return
	}

	re, err := regexp.Compile(rp.pattern)
	if err != nil || rp.pattern == "" {
		rp.testResultBox.Hide()
		return
	}

	rp.testResultBox.Show()

	if re.MatchString(testURL) {
		rp.testResult.Text = "✓ " + i18n.T("config.rules_regex_match")
		rp.testResult.Color = ui.ColorSuccess
	} else {
		rp.testResult.Text = "✗ " + i18n.T("config.rules_regex_no_match")
		rp.testResult.Color = ui.ColorDanger
	}

	rp.testResult.Refresh()
}

// withSubtleBackground wraps a widget in a container with a very subtle
// background tint, used for alternating row colors in rule lists.
func withSubtleBackground(obj fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(ui.ColorAltRowBg)
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
