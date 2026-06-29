//go:build darwin

package darwin

import (
	"os"
	"path/filepath"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.PathProvider = (*PathProvider)(nil)

type PathProvider struct{}

func (p *PathProvider) GetConfigFolderPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Application Support", "linkquisition")
	}
	return filepath.Join(configDir, "linkquisition")
}

func (p *PathProvider) GetLogFolderPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "linkquisition")
	}
	return filepath.Join(homeDir, "Library", "Logs", "linkquisition")
}

func (p *PathProvider) GetPluginFolderPath() string {
	execPath, err := os.Executable()
	if err == nil {
		bundlePath := filepath.Join(filepath.Dir(execPath), "..", "Resources", "plugins")
		if _, statErr := os.Stat(bundlePath); statErr == nil {
			return bundlePath
		}
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "linkquisition", "plugins")
}
