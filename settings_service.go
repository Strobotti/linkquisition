package linkquisition

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configDirPerms = 0700
const configFilePerms = 0600

// PathProvider supplies platform-specific paths used by the SettingsService.
type PathProvider interface {
	GetConfigFolderPath() string
	GetLogFolderPath() string
	GetPluginFolderPath() string
}

// FileSettingsService is the shared, platform-independent implementation of SettingsService.
type FileSettingsService struct {
	BrowserService BrowserService
	PathProvider   PathProvider
}

var _ SettingsService = (*FileSettingsService)(nil)

func (s *FileSettingsService) GetConfigFilePath() string {
	return filepath.Join(s.PathProvider.GetConfigFolderPath(), "config.json")
}

func (s *FileSettingsService) GetLogFilePath() string {
	return filepath.Join(s.PathProvider.GetLogFolderPath(), "linkquisition.log")
}

func (s *FileSettingsService) GetLogFolderPath() string {
	return s.PathProvider.GetLogFolderPath()
}

func (s *FileSettingsService) GetPluginFolderPath() string {
	return s.PathProvider.GetPluginFolderPath()
}

func (s *FileSettingsService) ReadSettings() (*Settings, error) {
	data, err := os.ReadFile(s.GetConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("unable to open config-file `%s` for reading: %v", s.GetConfigFilePath(), err)
	}

	var settings = &Settings{}
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("unable to parse the config-file `%s`: %v", s.GetConfigFilePath(), err)
	}

	settings.CompileAllRegexMatches()

	return settings, nil
}

func (s *FileSettingsService) WriteSettings(settings *Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal settings: %v", err)
	}

	if errMkdir := os.MkdirAll(s.PathProvider.GetConfigFolderPath(), os.FileMode(configDirPerms)); errMkdir != nil {
		return fmt.Errorf("failed to write settings: %v", errMkdir)
	}

	if errWrite := os.WriteFile(s.GetConfigFilePath(), data, os.FileMode(configFilePerms)); errWrite != nil {
		return fmt.Errorf("failed to write settings: %v", errWrite)
	}

	return nil
}

func (s *FileSettingsService) IsConfigured() (bool, error) {
	if _, err := os.Stat(s.GetConfigFilePath()); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	_, err := s.ReadSettings()

	return err == nil, err
}

func (s *FileSettingsService) GetSettings() *Settings {
	isConfigured, err := s.IsConfigured()
	if !isConfigured || err != nil {
		return GetDefaultSettings()
	}

	settings, err := s.ReadSettings()
	if err != nil {
		return GetDefaultSettings()
	}

	return settings
}

func (s *FileSettingsService) ScanBrowsers() error {
	var oldSettings *Settings

	if isConfigured, configErr := s.IsConfigured(); !isConfigured || configErr != nil {
		oldSettings = &Settings{}
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

	if err := s.WriteSettings(newSettings); err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	return nil
}
