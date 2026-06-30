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

	LogLevelInfo = "info"
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

// MatchesUrl returns true if the given url matches any of the browser's rules
func (s *BrowserSettings) MatchesUrl(u string) bool {
	uu := NewURL(u)

	for i := range s.Matches {
		switch s.Matches[i].Type {
		case BrowserMatchTypeRegex:
			if re, err := regexp.Compile(s.Matches[i].Value); err == nil && re.MatchString(u) {
				return true
			}
		case BrowserMatchTypeDomain:
			if domain, err := uu.GetDomain(); err == nil {
				if strings.EqualFold(domain, s.Matches[i].Value) {
					return true
				}
			}
		case BrowserMatchTypeSite:
			match := siteRegex.FindStringSubmatch(u)
			if len(match) > 1 && strings.EqualFold(match[1], s.Matches[i].Value) {
				return true
			}
		}
	}

	return false
}

type PluginSettings struct {
	// Path is the path to the plugin binary
	Path string `json:"path"`

	// IsDisabled allows temporarily disabling individual plugins
	IsDisabled bool `json:"isDisabled"`

	Settings map[string]interface{} `json:"settings,omitempty"`
}

type UiSettings struct {
	HideKeyboardGuideLabel bool `json:"hideKeyboardGuideLabel,omitempty"`
}

type Settings struct {
	LogLevel string            `json:"logLevel,omitempty"`
	Browsers []BrowserSettings `json:"browsers"`
	Plugins  []PluginSettings  `json:"plugins,omitempty"`
	Ui       UiSettings        `json:"ui,omitempty"`
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

	//nolint:gocritic // appendAssign: intentionally building a new combined slice
	s.Browsers = append(visibleBrowsers, hiddenBrowsers...)

	return s
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

	s.Browsers = browserSettings

	return s
}

func (s *Settings) addMissingBrowsers(browsers []Browser) *Settings {
	for i := range browsers {
		found := false
		for j := range s.Browsers {
			if s.Browsers[j].Command == browsers[i].Command {
				found = true
				break
			}
		}

		if !found {
			s.Browsers = append(
				s.Browsers, BrowserSettings{
					Name:    browsers[i].Name,
					Command: browsers[i].Command,
					Hidden:  false,
					Source:  SourceAuto,
				},
			)
		}
	}

	return s
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

func (s *Settings) AddRuleToBrowser(b *Browser, matchType, matchValue string) {
	for i := range s.Browsers {
		if s.Browsers[i].Command == b.Command {
			s.Browsers[i].Matches = append(
				s.Browsers[i].Matches,
				BrowserMatch{
					Type:  matchType,
					Value: matchValue,
				},
			)
		}
	}
}

type SettingsService interface {
	// IsConfigured returns true if the settings have been configured (i.e. the config-file exists)
	IsConfigured() (bool, error)

	// GetSettings returns the settings, either from the config-file or the default settings
	GetSettings() *Settings

	// ReadSettings reads the config-file and returns the settings
	ReadSettings() (*Settings, error)

	// WriteSettings writes the settings to the config-file
	WriteSettings(settings *Settings) error

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
		LogLevel: LogLevelInfo,
		Browsers: nil,
		Ui:       UiSettings{},
	}
}

func MapSettingsLogLevelToSlog(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
