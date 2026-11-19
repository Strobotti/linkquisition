//go:build !windows

package main

import (
	"errors"
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

type testHealthChecker struct {
	settings  map[string]interface{}
	configDir string
	err       error
}

func (checker *testHealthChecker) HealthCheck(settings map[string]interface{}, configDir string) ([]linkquisition.PluginHealthIssue, error) {
	checker.settings = settings
	checker.configDir = configDir
	return []linkquisition.PluginHealthIssue{{Message: "Plugin-provided issue"}}, checker.err
}

func TestOptionalPluginHealthCheckDispatch(t *testing.T) {
	checker := &testHealthChecker{}
	settings := map[string]interface{}{"option": "value"}
	issues, err := checkPluginHealth(checker, settings, "config-dir")
	require.NoError(t, err)
	require.Equal(t, []linkquisition.PluginHealthIssue{{Message: "Plugin-provided issue"}}, issues)
	require.Equal(t, settings, checker.settings)
	require.Equal(t, "config-dir", checker.configDir)
	checker.err = errors.New("health check failed")
	_, err = checkPluginHealth(checker, settings, "config-dir")
	require.ErrorIs(t, err, checker.err)
	issues, err = checkPluginHealth(struct{}{}, settings, "config-dir")
	require.NoError(t, err)
	require.Empty(t, issues)
}

func TestHealthBannerOpensHealthTab(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	i18n.Init("en")
	c := &Configurator{fapp: app}
	generalTab := container.NewTabItem("General", widget.NewLabel("General"))
	otherTab := container.NewTabItem("Other", widget.NewLabel("Other"))
	c.healthTab = container.NewTabItem("Health", widget.NewLabel("Health"))
	c.tabs = container.NewAppTabs(generalTab, otherTab, c.healthTab)
	banner := c.buildHealthBanner()
	button, ok := banner.Objects[2].(*widget.Button)
	require.True(t, ok)
	require.Equal(t, generalTab, c.tabs.Selected())
	test.Tap(button)
	require.Equal(t, c.healthTab, c.tabs.Selected())
}

func TestHealthTabShowsScrollableIssues(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	i18n.Init("en")
	svc := newTestSettingsService(t)
	require.NoError(t, svc.WriteSettings(&linkquisition.Settings{Plugins: []linkquisition.PluginSettings{{Path: "example.so"}}}))
	c := &Configurator{fapp: app, settingsService: svc, healthBanner: container.NewVBox()}
	const issueCount = 12
	c.healthProbe = func(_ linkquisition.PluginSettings, _ string) ([]linkquisition.PluginHealthIssue, error) {
		issues := make([]linkquisition.PluginHealthIssue, issueCount)
		for i := range issues {
			issues[i].Message = fmt.Sprintf("Issue %d", i+1)
		}
		return issues, nil
	}
	content, ok := c.buildConfigurationHealthTab().(*fyne.Container)
	require.True(t, ok)
	var scroll *container.Scroll
	for _, object := range content.Objects {
		if candidate, isScroll := object.(*container.Scroll); isScroll {
			scroll = candidate
			break
		}
	}
	require.NotNil(t, scroll)
	require.Equal(t, fyne.ScrollVerticalOnly, scroll.Direction)
	list, ok := scroll.Content.(*fyne.Container)
	require.True(t, ok)
	require.Len(t, list.Objects, issueCount*2-1)
	last, ok := list.Objects[len(list.Objects)-1].(*fyne.Container)
	require.True(t, ok)
	label, ok := last.Objects[0].(*widget.Label)
	require.True(t, ok)
	require.Equal(t, "Issue 12", label.Text)
	window := test.NewWindow(content)
	defer window.Close()
	window.Resize(fyne.NewSize(400, 180))
	scroll.ScrollToBottom()
	require.Greater(t, scroll.Offset.Y, float32(0))
}

func TestConfigurationHealthBannerRefreshes(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	i18n.Init("en")
	svc := newTestSettingsService(t)
	settings := &linkquisition.Settings{Plugins: []linkquisition.PluginSettings{{
		Path: "example.so", Settings: map[string]interface{}{"issue": true},
	}}}
	require.NoError(t, svc.WriteSettings(settings))
	c := &Configurator{fapp: app, settingsService: svc, healthBanner: container.NewVBox()}
	c.healthProbe = func(plugin linkquisition.PluginSettings, _ string) ([]linkquisition.PluginHealthIssue, error) {
		if plugin.Settings["issue"] == true {
			return []linkquisition.PluginHealthIssue{{Message: "Plugin issue"}}, nil
		}
		return nil, nil
	}
	c.healthBanner.Hide()
	c.buildConfigurationHealthTab()
	require.True(t, c.healthBanner.Visible())
	settings.Plugins[0].Settings["issue"] = false
	require.NoError(t, svc.WriteSettings(settings))
	c.healthRefresh()
	require.False(t, c.healthBanner.Visible())
}

func TestConfigurationHealthUsesEnabledPluginChecks(t *testing.T) {
	svc := newTestSettingsService(t)
	settings := &linkquisition.Settings{Plugins: []linkquisition.PluginSettings{
		{Path: "first.so"},
		{Path: "disabled.so", IsDisabled: true},
		{Path: "unsupported.so"},
		{Path: "failed.so"},
	}}
	require.NoError(t, svc.WriteSettings(settings))
	var checked []string
	c := &Configurator{settingsService: svc}
	c.healthProbe = func(plugin linkquisition.PluginSettings, configDir string) ([]linkquisition.PluginHealthIssue, error) {
		require.Equal(t, svc.GetConfigFolderPath(), configDir)
		checked = append(checked, plugin.Path)
		switch plugin.Path {
		case "first.so":
			return []linkquisition.PluginHealthIssue{{Message: "First issue"}, {Message: "Second issue"}}, nil
		case "failed.so":
			return nil, errors.New("check failed")
		default:
			return nil, nil
		}
	}
	require.Equal(t, []configurationHealthIssue{
		{pluginIndex: 0, message: "First issue"},
		{pluginIndex: 0, message: "Second issue"},
		{pluginIndex: 3, message: "failed.so", checkFailed: true},
	}, c.configurationHealth())
	require.Equal(t, []string{"first.so", "unsupported.so", "failed.so"}, checked)

	previous := noPlugins
	noPlugins = true
	t.Cleanup(func() { noPlugins = previous })
	checked = nil
	require.Empty(t, c.configurationHealth())
	require.Empty(t, checked)
}
