//go:build linux

package freedesktop

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
		return ".config"
	}

	return filepath.Join(configDir, "linkquisition")
}

func (p *PathProvider) GetLogFolderPath() string {
	stateDir, isset := os.LookupEnv("XDG_STATE_HOME")
	if isset {
		return filepath.Join(stateDir, "linkquisition")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(homeDir, ".local", "state", "linkquisition")
	}

	return filepath.Join(os.TempDir(), "linkquisition")
}

func (p *PathProvider) GetPluginFolderPath() string {
	return "/usr/lib/linkquisition/plugins"
}
