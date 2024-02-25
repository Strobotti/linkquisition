package freedesktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/strobotti/linkquisition"
)

var configDirPerms = 0700

var _ linkquisition.SettingsService = (*SettingsService)(nil)

type SettingsService struct {
	BrowserService linkquisition.BrowserService
}

func (s *SettingsService) GetConfigFilePath() string {
	return filepath.Join(s.GetConfigFolderPath(), "config.json")
}

func (s *SettingsService) GetConfigFolderPath() string {
	// get the user's home directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ".config"
	}

	return filepath.Join(configDir, "linkquisition")
}

func (s *SettingsService) GetLogFilePath() string {
	return filepath.Join(s.GetLogFolderPath(), "linkquisition.log")
}

func (s *SettingsService) GetLogFolderPath() string {
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

func (s *SettingsService) GetPluginFolderPath() string {
	return "/usr/lib/linkquisition/plugins"
}

func (s *SettingsService) ReadSettings() (*linkquisition.Settings, error) {
	data, err := os.ReadFile(s.GetConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("unable to open config-file `%s` for reading: %v", s.GetConfigFilePath(), err)
	}

	var settings = &linkquisition.Settings{}

	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("unable to parse the config-file `%s`: %v", s.GetConfigFilePath(), err)
	}

	return settings, nil
}

func (s *SettingsService) IsConfigured() bool {
	_, err := os.Stat(s.GetConfigFilePath())

	return !errors.Is(err, os.ErrNotExist)
}

func (s *SettingsService) GetSettings() *linkquisition.Settings {
	if !s.IsConfigured() {
		return linkquisition.GetDefaultSettings()
	}

	settings, err := s.ReadSettings()
	if err != nil {
		return linkquisition.GetDefaultSettings()
	}

	return settings
}

func (s *SettingsService) ScanBrowsers() error {
	var oldSettings *linkquisition.Settings

	if !s.IsConfigured() {
		oldSettings = &linkquisition.Settings{}
	} else {
		var err error
		if oldSettings, err = s.ReadSettings(); err != nil {
			return fmt.Errorf("failed to scan browsers: %v", err)
		}
	}

	browsers, err := s.BrowserService.GetAvailableBrowsers()
	if err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	newSettings := oldSettings.UpdateWithBrowsers(browsers).NormalizeBrowsers()

	// ensure the directory exists
	if errMkdir := os.MkdirAll(s.GetConfigFolderPath(), os.FileMode(configDirPerms)); errMkdir != nil {
		return fmt.Errorf("failed to scan browsers: %v", errMkdir)
	}

	data, err := json.MarshalIndent(newSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	//nolint:gomnd
	if err := os.WriteFile(s.GetConfigFilePath(), data, 0600); err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	return nil
}
