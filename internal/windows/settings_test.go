//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathProvider_GetConfigFolderPath(t *testing.T) {
	p := &PathProvider{}
	path := p.GetConfigFolderPath()

	assert.NotEmpty(t, path)
	assert.True(t, strings.HasSuffix(path, filepath.Join("linkquisition")))
	// Should be under APPDATA
	appData := os.Getenv("APPDATA")
	if appData != "" {
		assert.True(t, strings.HasPrefix(path, appData))
	}
}

func TestPathProvider_GetLogFolderPath(t *testing.T) {
	p := &PathProvider{}
	path := p.GetLogFolderPath()

	assert.NotEmpty(t, path)
	assert.True(t, strings.HasSuffix(path, filepath.Join("linkquisition", "logs")))
}

func TestPathProvider_GetPluginFolderPath(t *testing.T) {
	p := &PathProvider{}
	path := p.GetPluginFolderPath()

	assert.Empty(t, path, "plugins are not supported on Windows")
}
