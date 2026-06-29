//go:build darwin

package darwin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/strobotti/linkquisition"
)

var configDirPerms = 0700
var configFilePerms = 0600

var _ linkquisition.SettingsService = (*SettingsService)(nil)

type SettingsService struct {
	BrowserService linkquisition.BrowserService
}

func (s *SettingsService) GetConfigFilePath() string {
	return filepath.Join(s.GetConfigFolderPath(), "config.json")
}

func (s *SettingsService) GetConfigFolderPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Application Support", "linkquisition")
	}
	return filepath.Join(configDir, "linkquisition")
}

func (s *SettingsService) GetLogFilePath() string {
	return filepath.Join(s.GetLogFolderPath(), "linkquisition.log")
}

func (s *SettingsService) GetLogFolderPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "linkquisition")
	}
	return filepath.Join(homeDir, "Library", "Logs", "linkquisition")
}

func (s *SettingsService) GetPluginFolderPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "linkquisition", "plugins")
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

func (s *SettingsService) WriteSettings(settings *linkquisition.Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal settings: %v", err)
	}

	if errMkdir := os.MkdirAll(s.GetConfigFolderPath(), os.FileMode(configDirPerms)); errMkdir != nil {
		return fmt.Errorf("failed to write settings: %v", errMkdir)
	}

	if errWrite := os.WriteFile(s.GetConfigFilePath(), data, os.FileMode(configFilePerms)); errWrite != nil {
		return fmt.Errorf("failed to write settings: %v", errWrite)
	}

	return nil
}

func (s *SettingsService) IsConfigured() (bool, error) {
	if _, err := os.Stat(s.GetConfigFilePath()); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	_, err := s.ReadSettings()
	return err == nil, err
}

func (s *SettingsService) GetSettings() *linkquisition.Settings {
	isConfigured, err := s.IsConfigured()
	if !isConfigured || err != nil {
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

	if isConfigured, configErr := s.IsConfigured(); !isConfigured || configErr != nil {
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

	if errMkdir := os.MkdirAll(s.GetConfigFolderPath(), os.FileMode(configDirPerms)); errMkdir != nil {
		return fmt.Errorf("failed to scan browsers: %v", errMkdir)
	}

	data, err := json.MarshalIndent(newSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	if err := os.WriteFile(s.GetConfigFilePath(), data, os.FileMode(configFilePerms)); err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	return nil
}
