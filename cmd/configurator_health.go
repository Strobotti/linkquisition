package main

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition/internal/confighealth"
	"github.com/strobotti/linkquisition/internal/i18n"
)

// buildConfigurationHealthSection only checks local settings; it never downloads lists.
func (c *Configurator) buildConfigurationHealthSection() fyne.CanvasObject {
	results := container.NewVBox()
	title := widget.NewLabel(i18n.T("config.health_title"))
	title.TextStyle.Bold = true
	c.healthRefresh = func() {
		issues := confighealth.Check(c.settingsService.GetSettings(), runtime.GOOS != osWindows, c.settingsService.GetConfigFolderPath())
		results.RemoveAll()
		if len(issues) == 0 {
			results.Add(widget.NewLabel(i18n.T("config.health_ok")))
			if c.healthBanner != nil {
				c.healthBanner.Hide()
			}
		} else {
			if c.healthBanner != nil {
				c.healthBanner.Show()
			}
			for _, issue := range issues {
				var text string
				switch issue.Code {
				case confighealth.IssueDefangURLhaus:
					text = i18n.T("config.health_urlhaus")
				case confighealth.IssueDefangSourceGone:
					text = i18n.T("config.health_source_gone", map[string]interface{}{"Number": issue.SourceIndex + 1})
				default:
					continue
				}
				index := issue.PluginIndex
				message := widget.NewLabel(text)
				message.Wrapping = fyne.TextWrapWord
				button := widget.NewButton(i18n.T("config.health_edit_plugin"), func() { c.openHealthPluginSettings(index) })
				results.Add(container.NewVBox(message, button))
			}
		}
		results.Refresh()
	}
	check := widget.NewButton(i18n.T("config.health_check"), func() { c.healthRefresh() })
	c.healthRefresh()
	return container.NewVBox(title, results, check)
}
