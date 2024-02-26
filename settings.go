package linkquisition

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
)

var ErrNoMatchFound = errors.New("no match found")

const (
	BrowserMatchTypeRegex  = "regex"
	BrowserMatchTypeDomain = "domain"
	BrowserMatchTypeSite   = "site"

	SourceAuto   = "auto"
	SourceManual = "manual"
)

type BrowserMatch struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type BrowserSettings struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Hidden  bool   `json:"hidden"`
	Source  string `json:"source"`

	Matches []BrowserMatch `json:"matches"`
}

func (s *BrowserSettings) MatchesUrl(u string) bool {
	uu := NewURL(u)

	matchSite := func(u, site string) bool {
		re := regexp.MustCompile(`^https?://([^/]+)(/|$)`)
		match := re.FindStringSubmatch(u)
		if len(match) > 1 {
			return strings.EqualFold(match[1], site)
		}

		return false
	}

	for i := range s.Matches {
		switch s.Matches[i].Type {
		case BrowserMatchTypeRegex:
			if matches, _ := regexp.MatchString(s.Matches[i].Value, u); matches {
				return true
			}
		case BrowserMatchTypeDomain:
			if domain, err := uu.GetDomain(); err == nil {
				if strings.EqualFold(domain, s.Matches[i].Value) {
					return true
				}
			}
		case BrowserMatchTypeSite:
			if matchSite(u, s.Matches[i].Value) {
				return true
			}
		}
	}

	return false
}

type PluginSettings struct {
	// Path is the path to the plugin binary
	Path     string                 `json:"path"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

type Settings struct {
	LogLevel string            `json:"logLevel,omitempty"`
	Browsers []BrowserSettings `json:"browsers"`
	Plugins  []PluginSettings  `json:"plugins,omitempty"`
}

// NormalizeBrowsers moves hidden browsers to the end of the list
func (s *Settings) NormalizeBrowsers() *Settings {
	var visibleBrowsers []BrowserSettings
	var hiddenBrowsers []BrowserSettings

	for i := range s.Browsers {
		if !s.Browsers[i].Hidden {
			visibleBrowsers = append(visibleBrowsers, s.Browsers[i])
		} else {
			hiddenBrowsers = append(hiddenBrowsers, s.Browsers[i])
		}
	}

	var normalizedSettings = &Settings{
		Browsers: []BrowserSettings{},
	}

	normalizedSettings.Browsers = append(normalizedSettings.Browsers, visibleBrowsers...)
	normalizedSettings.Browsers = append(normalizedSettings.Browsers, hiddenBrowsers...)

	return normalizedSettings
}

func (s *Settings) UpdateWithBrowsers(browsers []Browser) *Settings {
	return s.dropAutoAddedBrowsersNoLongerPresent(browsers).addMissingBrowsers(browsers).NormalizeBrowsers()
}

func (s *Settings) dropAutoAddedBrowsersNoLongerPresent(browsers []Browser) *Settings {
	var browserSettings []BrowserSettings

	for i := range s.Browsers {
		if s.Browsers[i].Source == SourceManual {
			browserSettings = append(browserSettings, s.Browsers[i])
			continue
		}

		for j := range browsers {
			if s.Browsers[i].Command == browsers[j].Command {
				browserSettings = append(browserSettings, s.Browsers[i])
				break
			}
		}
	}

	return &Settings{
		Browsers: browserSettings,
	}
}

func (s *Settings) addMissingBrowsers(browsers []Browser) *Settings {
	browserSettings := s.Browsers // we need to keep the order

	for i := range browsers {
		found := false
		for j := range s.Browsers {
			if s.Browsers[j].Command == browsers[i].Command {
				found = true
				break
			}
		}

		if !found {
			browserSettings = append(
				browserSettings, BrowserSettings{
					Name:    browsers[i].Name,
					Command: browsers[i].Command,
					Hidden:  false,
					Source:  SourceAuto,
				},
			)
		}
	}

	return &Settings{
		Browsers: browserSettings,
	}
}

func (s *Settings) GetSelectableBrowsers() []Browser {
	var browsers []Browser

	for i := range s.Browsers {
		if s.Browsers[i].Hidden {
			continue
		}

		browser := Browser{
			Name:    s.Browsers[i].Name,
			Command: s.Browsers[i].Command,
		}
		browsers = append(browsers, browser)
	}

	return browsers
}

func (s *Settings) GetMatchingBrowser(u string) (*Browser, error) {
	for i := range s.Browsers {
		if s.Browsers[i].MatchesUrl(u) {
			return &Browser{
				Name:    s.Browsers[i].Name,
				Command: s.Browsers[i].Command,
			}, nil
		}
	}

	return nil, ErrNoMatchFound
}

type SettingsService interface {
	// IsConfigured returns true if the settings have been configured (i.e. the config-file exists)
	IsConfigured() (bool, error)

	// GetSettings returns the settings, either from the config-file or the default settings
	GetSettings() *Settings

	// ReadSettings reads the config-file and returns the settings
	ReadSettings() (*Settings, error)

	// ScanBrowsers scans (or re-scans) the system for available browsers and creates/updates the config-file
	ScanBrowsers() error

	// GetLogFilePath returns the path to the config-file
	GetLogFilePath() string

	// GetLogFolderPath returns the path to the config-file
	GetLogFolderPath() string

	// GetPluginFolderPath returns the absolute path to the plugin-folder
	GetPluginFolderPath() string
}

func GetDefaultSettings() *Settings {
	return &Settings{
		LogLevel: "info",
		Browsers: nil,
	}
}

func MapSettingsLogLevelToSlog(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
