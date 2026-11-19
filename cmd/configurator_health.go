package main

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition/internal/i18n"
)

func (c *Configurator) buildHealthBanner() *fyne.Container {
	warningIcon := fyne.NewStaticResource("configuration-warning.svg", []byte(warningIconSVG))
	return container.NewHBox(
		widget.NewIcon(warningIcon),
		widget.NewLabel(i18n.T("config.health_banner")),
		widget.NewButton(i18n.T("config.health_review"), func() {
			if c.tabs != nil && c.healthTab != nil {
				c.tabs.Select(c.healthTab)
			}
		}),
	)
}

type configurationHealthIssue struct {
	pluginIndex int
	message     string
	checkFailed bool
}

// configurationHealth checks enabled plugins against the current saved settings.
func (c *Configurator) configurationHealth() []configurationHealthIssue {
	if noPlugins {
		return nil
	}
	probe := c.healthProbe
	if probe == nil {
		probe = c.probePluginHealth
	}
	settings := c.settingsService.GetSettings()
	var results []configurationHealthIssue
	for index, plugin := range settings.Plugins {
		if plugin.IsDisabled {
			continue
		}
		issues, err := probe(plugin, c.settingsService.GetConfigFolderPath())
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("Plugin health check failed", "plugin", plugin.Path, "error", err)
			}
			results = append(results, configurationHealthIssue{pluginIndex: index, message: filepath.Base(plugin.Path), checkFailed: true})
			continue
		}
		for _, issue := range issues {
			results = append(results, configurationHealthIssue{pluginIndex: index, message: issue.Message})
		}
	}
	return results
}

// buildConfigurationHealthTab only checks local settings; it never downloads lists.
func (c *Configurator) buildConfigurationHealthTab() fyne.CanvasObject {
	results := container.NewVBox()
	title := widget.NewLabel(i18n.T("config.health_title"))
	title.TextStyle.Bold = true
	c.healthRefresh = func() {
		issues := c.configurationHealth()
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
			for i, issue := range issues {
				if i > 0 {
					results.Add(widget.NewSeparator())
				}
				text := issue.message
				if issue.checkFailed {
					text = i18n.T("config.health_check_failed", map[string]interface{}{templateKeyName: issue.message})
				}
				index := issue.pluginIndex
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
	return container.NewBorder(title, container.NewHBox(check), nil, nil, container.NewVScroll(results))
}
