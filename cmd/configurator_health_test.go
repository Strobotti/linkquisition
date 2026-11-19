//go:build !windows

package main

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/defangconfig"
	"github.com/strobotti/linkquisition/internal/i18n"
)

func TestConfigurationHealthBannerRefreshes(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	i18n.Init("en")
	svc := newTestSettingsService(t)
	settings := &linkquisition.Settings{Plugins: []linkquisition.PluginSettings{{
		Path:     "defang.so",
		Settings: map[string]interface{}{"sources": []string{defangconfig.URLhausHostFile}},
	}}}
	require.NoError(t, svc.WriteSettings(settings))
	c := &Configurator{fapp: app, settingsService: svc, healthBanner: container.NewVBox()}
	c.healthBanner.Hide()
	c.buildConfigurationHealthSection()
	require.True(t, c.healthBanner.Visible())
	settings.Plugins[0].Settings = map[string]interface{}{"sources": []string{"https://example.test/hosts"}}
	require.NoError(t, svc.WriteSettings(settings))
	c.healthRefresh()
	require.False(t, c.healthBanner.Visible())
}
