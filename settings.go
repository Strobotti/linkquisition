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

	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

type BrowserMatch struct {
	Type  string `json:"type"`
	Value string `json:"value"`

	// compiledRegex caches the compiled regex pattern for BrowserMatchTypeRegex matches.
	// Populated by CompileRegexMatches to avoid recompilation on every URL match.
	compiledRegex *regexp.Regexp
}

type BrowserSettings struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Hidden  bool   `json:"hidden"`
	Source  string `json:"source"`

	Matches []BrowserMatch `json:"matches"`
}

// CompileRegexMatches pre-compiles all regex match patterns for this browser.
// Invalid patterns are logged and will be skipped during matching.
func (s *BrowserSettings) CompileRegexMatches() {
	for i := range s.Matches {
		if s.Matches[i].Type == BrowserMatchTypeRegex {
			if re, err := regexp.Compile(s.Matches[i].Value); err == nil {
				s.Matches[i].compiledRegex = re
			} else {
				slog.Warn("Invalid regex pattern in browser match rule",
					"browser", s.Name, "pattern", s.Matches[i].Value, "error", err)
			}
		}
	}
}

// MatchesUrl returns true if the given url matches any of the browser's rules
func (s *BrowserSettings) MatchesUrl(u string) bool {
	uu := NewURL(u)

	for i := range s.Matches {
		switch s.Matches[i].Type {
		case BrowserMatchTypeRegex:
			re := s.Matches[i].compiledRegex
			if re == nil {
				// Fallback: try to compile if not pre-compiled (e.g. dynamically added rule)
				var err error
				re, err = regexp.Compile(s.Matches[i].Value)
				if err != nil {
					continue
				}
				s.Matches[i].compiledRegex = re
			}
			if re.MatchString(u) {
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
	Locale   string            `json:"locale,omitempty"`
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

// CompileAllRegexMatches pre-compiles regex patterns for all browsers.
// Call after loading settings to avoid repeated compilation during URL matching.
func (s *Settings) CompileAllRegexMatches() *Settings {
	for i := range s.Browsers {
		s.Browsers[i].CompileRegexMatches()
	}
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

	// GetLogFilePath returns the absolute path to the log-file
	GetLogFilePath() string

	// GetLogFolderPath returns the absolute path to the log-folder
	GetLogFolderPath() string

	// GetPluginFolderPath returns the absolute path to the plugin-folder
	GetPluginFolderPath() string

	// GetConfigFilePath returns the absolute path to the config-file
	GetConfigFilePath() string

	// GetConfigFolderPath returns the absolute path to the config-folder
	GetConfigFolderPath() string
}

func GetDefaultSettings() *Settings {
	return &Settings{
		LogLevel: LogLevelWarn,
		Browsers: nil,
		Ui:       UiSettings{},
	}
}

func MapSettingsLogLevelToSlog(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
