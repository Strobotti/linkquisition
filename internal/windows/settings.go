//go:build windows

package windows

import (
	"os"
	"path/filepath"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.PathProvider = (*PathProvider)(nil)

// PathProvider supplies Windows-specific paths for config, logs, and plugins.
// Config lives in %APPDATA%\linkquisition, logs in %LOCALAPPDATA%\linkquisition\logs,
// and plugins are not supported on Windows (returns empty string).
type PathProvider struct{}

func (p *PathProvider) GetConfigFolderPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "AppData", "Roaming", "linkquisition")
	}
	return filepath.Join(configDir, "linkquisition")
}

func (p *PathProvider) GetLogFolderPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "AppData", "Local", "linkquisition", "logs")
	}
	return filepath.Join(cacheDir, "linkquisition", "logs")
}

// GetPluginFolderPath returns an empty string on Windows — plugins are not supported.
func (p *PathProvider) GetPluginFolderPath() string {
	return ""
}
