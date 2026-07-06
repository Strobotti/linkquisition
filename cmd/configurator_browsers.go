package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) getBrowsersTab() fyne.CanvasObject {
	content := container.NewVBox()
	c.rebuildBrowsersList(content)
	return container.NewVScroll(content)
}

func (c *Configurator) rebuildBrowsersList(content *fyne.Container) {
	content.RemoveAll()

	settings := c.settingsService.GetSettings()

	if len(settings.Browsers) == 0 {
		emptyLabel := widget.NewLabel(i18n.T("config.browsers_empty"))
		emptyLabel.Wrapping = fyne.TextWrapWord

		scanBtn := widget.NewButton(i18n.T("config.scan_browsers"), func() {
			c.scanBrowsersAndRebuild(content)
		})
		scanBtn.Importance = widget.HighImportance

		content.Add(emptyLabel)
		content.Add(layout.NewSpacer())
		content.Add(scanBtn)
		content.Refresh()
		return
	}

	for idx := range settings.Browsers {
		if idx > 0 {
			content.Add(widget.NewSeparator())
		}
		content.Add(c.buildBrowserCard(settings, idx, content))
	}

	// Action buttons at the bottom
	content.Add(layout.NewSpacer())

	scanBtn := widget.NewButton(i18n.T("config.rescan_browsers"), func() {
		c.scanBrowsersAndRebuild(content)
	})

	addBtn := widget.NewButton(i18n.T("config.browsers_add"), func() {
		c.showAddBrowserDialog(content)
	})

	content.Add(container.NewHBox(scanBtn, addBtn))
	content.Refresh()
}

func (c *Configurator) buildBrowserCard(
	settings *linkquisition.Settings, idx int, listContainer *fyne.Container,
) fyne.CanvasObject {
	b := settings.Browsers[idx]

	// Title
	title := widget.NewLabel(b.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Command
	command := widget.NewLabel(b.Command)
	command.TextStyle = fyne.TextStyle{Italic: true}
	command.Truncation = fyne.TextTruncateEllipsis

	// Source badge
	sourceText := i18n.T("config.browsers_source_auto")
	if b.Source == linkquisition.SourceManual {
		sourceText = i18n.T("config.browsers_source_manual")
	}
	sourceLabel := widget.NewLabel(sourceText)
	sourceLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Visible toggle
	visibleCheck := widget.NewCheck(i18n.T("config.browsers_show_in_picker"), func(checked bool) {
		s := c.settingsService.GetSettings()
		s.Browsers[idx].Hidden = !checked
		if err := c.settingsService.WriteSettings(s); err != nil {
			c.logger.Error("Error saving browser visibility", "error", err)
		}
	})
	visibleCheck.Checked = !b.Hidden

	// Reorder buttons
	upBtn := widget.NewButton(i18n.T("config.plugins_move_up"), func() {
		c.moveBrowser(idx, -1, listContainer)
	})
	if idx == 0 {
		upBtn.Disable()
	}

	downBtn := widget.NewButton(i18n.T("config.plugins_move_down"), func() {
		c.moveBrowser(idx, 1, listContainer)
	})
	if idx == len(settings.Browsers)-1 {
		downBtn.Disable()
	}

	// Edit/Delete buttons for manual browsers
	var actionRow fyne.CanvasObject
	if b.Source == linkquisition.SourceManual {
		editBtn := widget.NewButton(i18n.T("config.browsers_edit"), func() {
			c.showEditBrowserDialog(idx, listContainer)
		})
		deleteBtn := widget.NewButton("🗑", func() {
			c.confirmDeleteBrowser(idx, listContainer)
		})
		actionRow = container.NewHBox(editBtn, deleteBtn)
	} else {
		actionRow = container.NewHBox()
	}

	// Rules count indicator
	rulesCount := len(b.Matches)
	var rulesLabel *widget.Label
	if rulesCount > 0 {
		rulesLabel = widget.NewLabel(i18n.T("config.browsers_rules_count", map[string]interface{}{
			"Count": rulesCount,
		}))
		rulesLabel.TextStyle = fyne.TextStyle{Italic: true}
	}

	// Layout
	headerRow := container.NewBorder(
		nil, nil,
		container.NewHBox(title, sourceLabel),
		container.NewHBox(upBtn, downBtn),
	)

	cardContent := container.NewVBox(headerRow, command)
	if rulesLabel != nil {
		cardContent.Add(rulesLabel)
	}
	cardContent.Add(container.NewHBox(visibleCheck, layout.NewSpacer(), actionRow))

	return widget.NewCard("", "", cardContent)
}

func (c *Configurator) moveBrowser(idx, direction int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	newIdx := idx + direction

	if newIdx < 0 || newIdx >= len(settings.Browsers) {
		return
	}

	settings.Browsers[idx], settings.Browsers[newIdx] = settings.Browsers[newIdx], settings.Browsers[idx]

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error saving browser order", "error", err)
		return
	}

	c.rebuildBrowsersList(listContainer)
}

func (c *Configurator) showAddBrowserDialog(listContainer *fyne.Container) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder(i18n.T("config.browsers_name_placeholder"))

	commandEntry := widget.NewEntry()
	commandEntry.SetPlaceHolder(i18n.T("config.browsers_command_placeholder"))

	form := widget.NewForm(
		widget.NewFormItem(i18n.T("config.browsers_name"), nameEntry),
		widget.NewFormItem(i18n.T("config.browsers_command"), commandEntry),
	)

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	d := dialog.NewCustomConfirm(
		i18n.T("config.browsers_add_title"),
		i18n.T("config.plugins_save"),
		i18n.T("config.plugins_cancel"),
		form,
		func(save bool) {
			if !save {
				return
			}
			if nameEntry.Text == "" || commandEntry.Text == "" {
				return
			}
			c.addManualBrowser(nameEntry.Text, commandEntry.Text, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 200)) //nolint:mnd
	d.Show()
}

func (c *Configurator) addManualBrowser(name, command string, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()

	settings.Browsers = append(settings.Browsers, linkquisition.BrowserSettings{
		Name:    name,
		Command: command,
		Hidden:  false,
		Source:  linkquisition.SourceManual,
	})

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Error adding browser", "error", err, "name", name)
		return
	}

	c.rebuildBrowsersList(listContainer)
}

func (c *Configurator) showEditBrowserDialog(idx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	b := settings.Browsers[idx]

	nameEntry := widget.NewEntry()
	nameEntry.SetText(b.Name)

	commandEntry := widget.NewEntry()
	commandEntry.SetText(b.Command)

	form := widget.NewForm(
		widget.NewFormItem(i18n.T("config.browsers_name"), nameEntry),
		widget.NewFormItem(i18n.T("config.browsers_command"), commandEntry),
	)

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	d := dialog.NewCustomConfirm(
		i18n.T("config.browsers_edit_title"),
		i18n.T("config.plugins_save"),
		i18n.T("config.plugins_cancel"),
		form,
		func(save bool) {
			if !save {
				return
			}
			if nameEntry.Text == "" || commandEntry.Text == "" {
				return
			}
			s := c.settingsService.GetSettings()
			s.Browsers[idx].Name = nameEntry.Text
			s.Browsers[idx].Command = commandEntry.Text
			if err := c.settingsService.WriteSettings(s); err != nil {
				c.logger.Error("Error saving browser edit", "error", err)
				return
			}
			c.rebuildBrowsersList(listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 200)) //nolint:mnd
	d.Show()
}

func (c *Configurator) confirmDeleteBrowser(idx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	b := settings.Browsers[idx]

	windows := c.fapp.Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	parentWindow := windows[0]

	dialog.ShowConfirm(
		i18n.T("config.browsers_delete"),
		i18n.T("config.browsers_delete_confirm", map[string]interface{}{templateKeyName: b.Name}),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			s := c.settingsService.GetSettings()
			s.Browsers = append(s.Browsers[:idx], s.Browsers[idx+1:]...)
			if err := c.settingsService.WriteSettings(s); err != nil {
				c.logger.Error("Error deleting browser", "error", err)
				return
			}
			c.rebuildBrowsersList(listContainer)
		},
		parentWindow,
	)
}

func (c *Configurator) scanBrowsersAndRebuild(listContainer *fyne.Container) {
	go func() {
		if err := c.settingsService.ScanBrowsers(); err != nil {
			c.logger.Error("Error scanning browsers", "error", err)
			return
		}
		c.rebuildBrowsersList(listContainer)
	}()
}
