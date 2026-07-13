package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) getBrowsersTab() fyne.CanvasObject {
	content := container.NewVBox()
	c.rebuildBrowsersList(content)

	scrollArea := container.NewVScroll(content)

	scanBtn := widget.NewButton(i18n.T("config.rescan_browsers"), nil)
	scanBtn.OnTapped = func() {
		c.scanBrowsersAndRebuild(content, scanBtn)
	}

	addBtn := widget.NewButton(i18n.T("config.browsers_add"), func() {
		c.showAddBrowserDialog(content)
	})

	// Legend with hidden-eye icon explaining visibility behavior
	legendIcon := widget.NewIcon(theme.VisibilityOffIcon())
	legendLabel := widget.NewLabel(i18n.T("config.browsers_visibility_legend"))
	legendLabel.TextStyle = fyne.TextStyle{Italic: true}
	legendLabel.Wrapping = fyne.TextWrapWord
	legendRow := container.NewBorder(nil, nil, legendIcon, nil, legendLabel)

	bottomBar := container.NewVBox(
		legendRow,
		container.NewHBox(scanBtn, addBtn),
	)

	return container.NewBorder(nil, bottomBar, nil, nil, scrollArea)
}

func (c *Configurator) rebuildBrowsersList(content *fyne.Container) {
	content.RemoveAll()

	settings := c.settingsService.GetSettings()

	if len(settings.Browsers) == 0 {
		emptyLabel := widget.NewLabel(i18n.T("config.browsers_empty"))
		emptyLabel.Wrapping = fyne.TextWrapWord

		scanBtn := widget.NewButton(i18n.T("config.scan_browsers"), nil)
		scanBtn.OnTapped = func() {
			c.scanBrowsersAndRebuild(content, scanBtn)
		}
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

	content.Refresh()
}

func (c *Configurator) buildBrowserIconPreview(b *linkquisition.BrowserSettings) fyne.CanvasObject {
	browser := linkquisition.Browser{
		Name:     b.Name,
		Command:  b.Command,
		IconPath: b.IconPath,
	}
	iconBytes, err := c.browserService.GetIconForBrowser(browser)
	if err != nil || len(iconBytes) == 0 {
		return nil
	}
	iconRes := fyne.NewStaticResource("browser-icon.png", iconBytes)
	img := canvas.NewImageFromResource(iconRes)
	img.SetMinSize(fyne.NewSize(32, 32)) //nolint:mnd
	img.FillMode = canvas.ImageFillContain
	return img
}

func (c *Configurator) buildBrowserCard(
	settings *linkquisition.Settings, idx int, listContainer *fyne.Container,
) fyne.CanvasObject {
	b := settings.Browsers[idx]

	// Icon preview
	iconWidget := c.buildBrowserIconPreview(&b)

	// Title
	title := widget.NewLabel(b.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Source badge
	sourceText := i18n.T("config.browsers_source_auto")
	if b.Source == linkquisition.SourceManual {
		sourceText = i18n.T("config.browsers_source_manual")
	}
	sourceLabel := widget.NewLabel(sourceText)
	sourceLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Visibility toggle — eye icon
	visibilityBtn := c.buildVisibilityToggle(b.Hidden, idx, listContainer)

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

	// Edit button — only for manual browsers
	var editBtn *widget.Button
	if b.Source == linkquisition.SourceManual {
		editBtn = widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
			c.showEditBrowserDialog(idx, listContainer)
		})
	}

	// Delete button — trash icon with confirmation
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		c.confirmDeleteBrowser(idx, listContainer)
	})

	// Command label
	command := widget.NewLabel(b.Command)
	command.TextStyle = fyne.TextStyle{Italic: true}
	command.Truncation = fyne.TextTruncateEllipsis

	// Rules count indicator (same line as command, right-aligned)
	rulesCount := len(b.Matches)
	var rulesLabel *widget.Label
	if rulesCount > 0 {
		rulesLabel = widget.NewLabel(i18n.T("config.browsers_rules_count", map[string]interface{}{
			"Count": rulesCount,
		}))
		rulesLabel.TextStyle = fyne.TextStyle{Italic: true}
	}

	// Layout — title row
	var titleRow fyne.CanvasObject
	if iconWidget != nil {
		titleRow = container.NewHBox(iconWidget, title, sourceLabel)
	} else {
		titleRow = container.NewHBox(title, sourceLabel)
	}

	// Right-side controls
	controls := container.NewHBox(visibilityBtn, upBtn, downBtn)
	if editBtn != nil {
		controls.Add(editBtn)
	}
	controls.Add(deleteBtn)

	headerRow := container.NewBorder(
		nil, nil,
		titleRow,
		controls,
	)

	// Command row with rules count on the right
	var commandRow fyne.CanvasObject
	if rulesLabel != nil {
		commandRow = container.NewBorder(nil, nil, nil, rulesLabel, command)
	} else {
		commandRow = command
	}

	cardContent := container.NewVBox(headerRow, commandRow)

	return widget.NewCard("", "", cardContent)
}

func (c *Configurator) buildVisibilityToggle(hidden bool, idx int, listContainer *fyne.Container) *widget.Button {
	btn := widget.NewButtonWithIcon("", theme.VisibilityIcon(), nil)
	if hidden {
		btn.Icon = theme.VisibilityOffIcon()
	}
	btn.OnTapped = func() {
		btn.Disable()
		go func() {
			s := c.settingsService.GetSettings()
			s.Browsers[idx].Hidden = !s.Browsers[idx].Hidden
			if err := c.settingsService.WriteSettings(s); err != nil {
				c.logger.Error("Error saving browser visibility", "error", err)
				fyne.Do(func() { btn.Enable() })
				return
			}
			fyne.Do(func() { c.rebuildBrowsersList(listContainer) })
		}()
	}
	return btn
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

	iconPathEntry := widget.NewEntry()
	iconPathEntry.SetPlaceHolder(i18n.T("config.browsers_icon_path_placeholder"))

	form := widget.NewForm(
		widget.NewFormItem(i18n.T("config.browsers_name"), nameEntry),
		widget.NewFormItem(i18n.T("config.browsers_command"), commandEntry),
		widget.NewFormItem(i18n.T("config.browsers_icon_path"), iconPathEntry),
	)

	parentWindow := c.parentWindow()
	if parentWindow == nil {
		return
	}

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
			c.addManualBrowser(nameEntry.Text, commandEntry.Text, iconPathEntry.Text, listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 250)) //nolint:mnd
	d.Show()
}

func (c *Configurator) addManualBrowser(name, command, iconPath string, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()

	settings.Browsers = append(settings.Browsers, linkquisition.BrowserSettings{
		Name:     name,
		Command:  command,
		IconPath: iconPath,
		Hidden:   false,
		Source:   linkquisition.SourceManual,
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

	iconPathEntry := widget.NewEntry()
	iconPathEntry.SetPlaceHolder(i18n.T("config.browsers_icon_path_placeholder"))
	iconPathEntry.SetText(b.IconPath)

	form := widget.NewForm(
		widget.NewFormItem(i18n.T("config.browsers_name"), nameEntry),
		widget.NewFormItem(i18n.T("config.browsers_command"), commandEntry),
		widget.NewFormItem(i18n.T("config.browsers_icon_path"), iconPathEntry),
	)

	parentWindow := c.parentWindow()
	if parentWindow == nil {
		return
	}

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
			s.Browsers[idx].IconPath = iconPathEntry.Text
			if err := c.settingsService.WriteSettings(s); err != nil {
				c.logger.Error("Error saving browser edit", "error", err)
				return
			}
			c.rebuildBrowsersList(listContainer)
		},
		parentWindow,
	)
	d.Resize(fyne.NewSize(500, 250)) //nolint:mnd
	d.Show()
}

func (c *Configurator) confirmDeleteBrowser(idx int, listContainer *fyne.Container) {
	settings := c.settingsService.GetSettings()
	b := settings.Browsers[idx]

	parentWindow := c.parentWindow()
	if parentWindow == nil {
		return
	}

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

func (c *Configurator) scanBrowsersAndRebuild(listContainer *fyne.Container, btn *widget.Button) {
	originalText := btn.Text
	btn.SetText(i18n.T("config.scan_browsers_scanning"))
	btn.Disable()

	go func() {
		if err := c.settingsService.ScanBrowsers(); err != nil {
			c.logger.Error("Error scanning browsers", "error", err)
			fyne.Do(func() {
				btn.SetText(originalText)
				btn.Enable()

				if pw := c.parentWindow(); pw != nil {
					dialog.ShowError(err, pw)
				}
			})
			return
		}
		fyne.Do(func() {
			btn.SetText(i18n.T("config.scan_browsers_done"))
			c.hideNoBrowsersWarning()
		})
		time.AfterFunc(time.Second, func() {
			fyne.Do(func() { c.rebuildBrowsersList(listContainer) })
		})
	}()
}
