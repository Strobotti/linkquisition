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
