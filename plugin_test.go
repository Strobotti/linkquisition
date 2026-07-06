package linkquisition_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/strobotti/linkquisition"
)

func TestNewPluginServiceProvider(t *testing.T) {
	logger := slog.Default()
	settings := &Settings{
		LogLevel: "debug",
		Browsers: []BrowserSettings{
			{Name: "Firefox", Command: "firefox"},
		},
	}

	provider := NewPluginServiceProvider(logger, settings, "/tmp/test-config")

	assert.Equal(t, logger, provider.GetLogger())
	assert.Equal(t, settings, provider.GetSettings())
	assert.Equal(t, "/tmp/test-config", provider.GetConfigFolderPath())
}

func TestNewPluginServiceProvider_NilLogger(t *testing.T) {
	settings := &Settings{}

	provider := NewPluginServiceProvider(nil, settings, "")

	assert.Nil(t, provider.GetLogger())
	assert.Equal(t, settings, provider.GetSettings())
}

func TestPluginResult_DefaultValues(t *testing.T) {
	result := PluginResult{}

	assert.Equal(t, ActionContinue, result.Action)
	assert.Equal(t, "", result.URL)
	assert.Equal(t, "", result.Message)
	assert.False(t, result.ContinueChain)
}

func TestPluginMetadata_SettingDescriptors(t *testing.T) {
	meta := PluginMetadata{
		Name:        "Test Plugin",
		Description: "A test plugin",
		Author:      "Test Author",
		Version:     "1.0.0",
		Settings: []PluginSettingDescriptor{
			{
				Key:         "timeout",
				Label:       "Request Timeout",
				Description: "How long to wait for HTTP requests",
				Type:        SettingTypeDuration,
				Default:     "5s",
			},
			{
				Key:     "action",
				Label:   "Action",
				Type:    SettingTypeChoice,
				Default: "block",
				Options: []string{"block", "warn", "log"},
			},
			{
				Key:      "sources",
				Label:    "Blocklist Sources",
				Type:     SettingTypeStringList,
				Required: true,
			},
		},
	}

	assert.Equal(t, "Test Plugin", meta.Name)
	assert.Len(t, meta.Settings, 3)
	assert.Equal(t, SettingTypeDuration, meta.Settings[0].Type)
	assert.Equal(t, SettingTypeChoice, meta.Settings[1].Type)
	assert.Equal(t, []string{"block", "warn", "log"}, meta.Settings[1].Options)
	assert.True(t, meta.Settings[2].Required)
}
