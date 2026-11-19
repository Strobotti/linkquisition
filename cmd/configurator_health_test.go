//go:build !windows

package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

type testHealthChecker struct {
	settings  map[string]interface{}
	configDir string
}

func (checker *testHealthChecker) HealthCheck(settings map[string]interface{}, configDir string) ([]linkquisition.PluginHealthIssue, error) {
	checker.settings = settings
	checker.configDir = configDir
	return []linkquisition.PluginHealthIssue{{Message: "Plugin-provided issue"}}, nil
}

func TestOptionalPluginHealthCheckDispatch(t *testing.T) {
	checker := &testHealthChecker{}
	settings := map[string]interface{}{"option": "value"}
	issues, err := checkPluginHealth(checker, settings, "config-dir")
	require.NoError(t, err)
	require.Equal(t, []linkquisition.PluginHealthIssue{{Message: "Plugin-provided issue"}}, issues)
	require.Equal(t, settings, checker.settings)
	require.Equal(t, "config-dir", checker.configDir)
	issues, err = checkPluginHealth(struct{}{}, settings, "config-dir")
	require.NoError(t, err)
	require.Empty(t, issues)
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
	c.buildConfigurationHealthSection()
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
