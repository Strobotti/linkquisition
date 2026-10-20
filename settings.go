package linkquisition

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
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

	PickerLayoutVertical   = "vertical"
	PickerLayoutHorizontal = "horizontal"

	ThemeSystem = "system"
	ThemeDark   = "dark"
	ThemeLight  = "light"

	FaviconStrategyDirect = "direct"
	FaviconStrategyParsed = "parsed"
	FaviconStrategyGoogle = "google"

	SecurityProviderGoogleSafeBrowsing = "google_safe_browsing"
	SecurityProviderVirusTotal         = "virustotal"

	DefaultSecurityCacheTTLHours = 24

	DefaultMaxItemsPerRow = 5
)

type BrowserMatch struct {
	Type  string `json:"type"`
	Value string `json:"value"`

	// compiledRegex caches the compiled regex pattern for BrowserMatchTypeRegex matches.
	// Populated by CompileRegexMatches to avoid recompilation on every URL match.
	compiledRegex *regexp.Regexp
}

type BrowserSettings struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	IconPath string `json:"iconPath,omitempty"`
	Hidden   bool   `json:"hidden"`
	Source   string `json:"source"`

	ProfileName      string `json:"profileName,omitempty"`
	ProfilePath      string `json:"profilePath,omitempty"`
	ProfileType      string `json:"profileType,omitempty"`
	ProfileIsDefault bool   `json:"profileIsDefault,omitempty"`

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
	HideKeyboardGuideLabel bool   `json:"hideKeyboardGuideLabel,omitempty"`
	PickerLayout           string `json:"pickerLayout,omitempty"`
	MaxItemsPerRow         int    `json:"maxItemsPerRow,omitempty"`
	Theme                  string `json:"theme,omitempty"`
	ShowFavicon            bool   `json:"showFavicon,omitempty"`
	FaviconStrategy        string `json:"faviconStrategy,omitempty"`
}

// GetPickerLayout returns the effective picker layout, defaulting to horizontal.
func (u *UiSettings) GetPickerLayout() string {
	if u.PickerLayout == PickerLayoutVertical {
		return PickerLayoutVertical
	}
	return PickerLayoutHorizontal
}

// GetMaxItemsPerRow returns the effective max items per row, defaulting to DefaultMaxItemsPerRow.
func (u *UiSettings) GetMaxItemsPerRow() int {
	if u.MaxItemsPerRow > 0 {
		return u.MaxItemsPerRow
	}
	return DefaultMaxItemsPerRow
}

// GetTheme returns the effective theme, defaulting to system.
func (u *UiSettings) GetTheme() string {
	switch u.Theme {
	case ThemeDark, ThemeLight:
		return u.Theme
	default:
		return ThemeSystem
	}
}

// GetFaviconStrategy returns the effective favicon retrieval strategy, defaulting to direct.
func (u *UiSettings) GetFaviconStrategy() string {
	switch u.FaviconStrategy {
	case FaviconStrategyParsed, FaviconStrategyGoogle:
		return u.FaviconStrategy
	default:
		return FaviconStrategyDirect
	}
}

// SecurityCacheSettings holds configuration for caching security check results.
type SecurityCacheSettings struct {
	Enabled  bool `json:"enabled,omitempty"`
	TTLHours int  `json:"ttlHours,omitempty"`
}

// GetTTL returns the effective cache TTL duration.
func (s *SecurityCacheSettings) GetTTL() time.Duration {
	if s.TTLHours > 0 {
		return time.Duration(s.TTLHours) * time.Hour
	}

	return DefaultSecurityCacheTTLHours * time.Hour
}

// SecuritySettings holds configuration for URL safety checking.
type SecuritySettings struct {
	Enabled  bool                  `json:"enabled,omitempty"`
	Provider string                `json:"provider,omitempty"`
	APIKey   string                `json:"apiKey,omitempty"`
	Cache    SecurityCacheSettings `json:"cache,omitempty"`
}

// GetProvider returns the effective security provider, defaulting to Google Safe Browsing.
func (s *SecuritySettings) GetProvider() string {
	if s.Provider == SecurityProviderVirusTotal {
		return SecurityProviderVirusTotal
	}

	return SecurityProviderGoogleSafeBrowsing
}

// IsConfigured returns true if security checking is enabled and an API key is set.
func (s *SecuritySettings) IsConfigured() bool {
	return s.Enabled && s.APIKey != ""
}

type Settings struct {
	Locale   string            `json:"locale,omitempty"`
	LogLevel string            `json:"logLevel,omitempty"`
	Browsers []BrowserSettings `json:"browsers"`
	Plugins  []PluginSettings  `json:"plugins,omitempty"`
	Ui       UiSettings        `json:"ui,omitempty"`
	Security SecuritySettings  `json:"security,omitempty"`
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
	s.migrateLegacyBrowserEntries(browsers)
	return s.dropAutoAddedBrowsersNoLongerPresent(browsers).
		addMissingBrowsers(browsers).
		refreshIconPaths(browsers).
		NormalizeBrowsers()
}

// migrateLegacyBrowserEntries attaches an existing unprofiled browser entry and its
// rules to the detected default profile when the old config predates profile targets.
func (s *Settings) migrateLegacyBrowserEntries(browsers []Browser) {
	for i := range s.Browsers {
		if s.Browsers[i].Source != SourceAuto || s.Browsers[i].ProfilePath != "" {
			continue
		}

		for _, browser := range browsers {
			if browser.Command != s.Browsers[i].Command || browser.ProfilePath == "" || !browser.ProfileIsDefault {
				continue
			}

			s.Browsers[i].Name = browser.Name
			s.Browsers[i].ProfileName = browser.ProfileName
			s.Browsers[i].ProfilePath = browser.ProfilePath
			s.Browsers[i].ProfileType = browser.ProfileType
			break
		}
	}
}

func (s *Settings) dropAutoAddedBrowsersNoLongerPresent(browsers []Browser) *Settings {
	var browserSettings []BrowserSettings

	for i := range s.Browsers {
		if s.Browsers[i].Source == SourceManual {
			browserSettings = append(browserSettings, s.Browsers[i])
			continue
		}

		for j := range browsers {
			if sameBrowserTarget(s.Browsers[i].Command, s.Browsers[i].ProfilePath, browsers[j].Command, browsers[j].ProfilePath) {
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
		if browsers[i].ProfilePath == "" && hasProfileTargets(browsers[i].Command, browsers) {
			continue
		}
		found := false
		for j := range s.Browsers {
			if sameBrowserTarget(s.Browsers[j].Command, s.Browsers[j].ProfilePath, browsers[i].Command, browsers[i].ProfilePath) ||
				(s.Browsers[j].Command == browsers[i].Command && s.Browsers[j].ProfilePath == "" && isDefaultBrowser(browsers[i])) {
				found = true
				break
			}
		}

		if !found {
			s.Browsers = append(
				s.Browsers, BrowserSettings{
					Name:             browsers[i].Name,
					Command:          browsers[i].Command,
					IconPath:         browsers[i].IconPath,
					Hidden:           false,
					Source:           SourceAuto,
					ProfileName:      browsers[i].ProfileName,
					ProfilePath:      browsers[i].ProfilePath,
					ProfileType:      browsers[i].ProfileType,
					ProfileIsDefault: browsers[i].ProfileIsDefault,
				},
			)
		}
	}

	return s
}

// refreshIconPaths updates the icon path for auto-discovered browsers that have
// a newer icon path available. Manual browsers keep their existing icon path unless
// it is empty and a new one was discovered.
func sameBrowserTarget(commandA, profileA, commandB, profileB string) bool {
	return commandA == commandB && profileA == profileB
}

func hasProfileTargets(command string, browsers []Browser) bool {
	for _, browser := range browsers {
		if browser.Command == command && browser.ProfilePath != "" {
			return true
		}
	}
	return false
}

func isDefaultBrowser(browser Browser) bool {
	return browser.ProfilePath == "" || browser.ProfileIsDefault
}

func (s *Settings) refreshIconPaths(browsers []Browser) *Settings {
	for i := range s.Browsers {
		if s.Browsers[i].ProfilePath == "" && hasProfileTargets(s.Browsers[i].Command, browsers) {
			continue
		}
		for j := range browsers {
			if !sameBrowserTarget(s.Browsers[i].Command, s.Browsers[i].ProfilePath, browsers[j].Command, browsers[j].ProfilePath) {
				continue
			}

			// Update icon path for auto browsers unconditionally,
			// for manual browsers only if currently empty.
			if s.Browsers[i].Source == SourceAuto || s.Browsers[i].IconPath == "" {
				if browsers[j].IconPath != "" {
					s.Browsers[i].IconPath = browsers[j].IconPath
				}
			}
			break
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
			Name:        s.Browsers[i].Name,
			Command:     s.Browsers[i].Command,
			IconPath:    s.Browsers[i].IconPath,
			ProfileName: s.Browsers[i].ProfileName,
			ProfilePath: s.Browsers[i].ProfilePath,
			ProfileType: s.Browsers[i].ProfileType,
		}
		browsers = append(browsers, browser)
	}

	return browsers
}

func (s *Settings) GetMatchingBrowser(u string) (*Browser, error) {
	for i := range s.Browsers {
		if !s.Browsers[i].MatchesUrl(u) {
			continue
		}
		return &Browser{
			Name:             s.Browsers[i].Name,
			Command:          s.Browsers[i].Command,
			IconPath:         s.Browsers[i].IconPath,
			ProfileName:      s.Browsers[i].ProfileName,
			ProfilePath:      s.Browsers[i].ProfilePath,
			ProfileType:      s.Browsers[i].ProfileType,
			ProfileIsDefault: s.Browsers[i].ProfileIsDefault,
		}, nil
	}

	return nil, ErrNoMatchFound
}

func (s *Settings) AddRuleToBrowser(b *Browser, matchType, matchValue string) {
	for i := range s.Browsers {
		if sameBrowserTarget(s.Browsers[i].Command, s.Browsers[i].ProfilePath, b.Command, b.ProfilePath) {
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
