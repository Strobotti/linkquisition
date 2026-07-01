package linkquisition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/strobotti/linkquisition"
)

func TestBrowserSettings_MatchesUrl(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		settings BrowserSettings
		url      string
		expected bool
	}{
		{
			name: "site matches",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeSite,
						Value: "www.example.com",
					},
				},
			},
			url:      "https://www.example.com/path/is/here",
			expected: true,
		},
		{
			name: "site does not match if the subdomain is different",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeSite,
						Value: "www.example.com",
					},
				},
			},
			url:      "https://example.com/path/is/here",
			expected: false,
		},
		{
			name: "domain matches",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeDomain,
						Value: "example.com",
					},
				},
			},
			url:      "https://example.com/path/is/here",
			expected: true,
		},
		{
			name: "domain matches, without path",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeDomain,
						Value: "example.com",
					},
				},
			},
			url:      "https://example.com",
			expected: true,
		},
		{
			name: "domain matches even if the subdomain is different",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeDomain,
						Value: "example.com",
					},
				},
			},
			url:      "https://sub.example.com/path/is/here",
			expected: true,
		},
		{
			name: "domain matches even if the subdomain is different, without path",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeDomain,
						Value: "example.com",
					},
				},
			},
			url:      "https://sub.example.com",
			expected: true,
		},
		{
			name: "domain does not match",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeDomain,
						Value: "example.com",
					},
				},
			},
			url:      "https://www.example.org/path/is/here",
			expected: false,
		},
		{
			name: "regex matches",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeRegex,
						Value: `^https?://github\.com/Strobotti/`,
					},
				},
			},
			url:      "https://github.com/Strobotti/linkquisition",
			expected: true,
		},
		{
			name: "regex does not match",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Hidden:  false,
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{
						Type:  BrowserMatchTypeRegex,
						Value: `^https?://github\.com/Strobotti/`,
					},
				},
			},
			url:      "https://github.com/",
			expected: false,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.settings.MatchesUrl(tt.url))
			},
		)
	}
}

func TestSettings_NormalizeBrowsers(t *testing.T) {
	for _, tt := range [...]struct {
		name             string
		inputBrowsers    []BrowserSettings
		expectedBrowsers []BrowserSettings
	}{
		{
			name: "no hidden browsers causes no changes",
			inputBrowsers: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
			expectedBrowsers: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
		},
		{
			name: "one hidden browser is moved to the end",
			inputBrowsers: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  true,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
			expectedBrowsers: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  true,
					Source:  SourceAuto,
				},
			},
		},
		{
			name: "only hidden browsers causes no changes",
			inputBrowsers: []BrowserSettings{
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  true,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  true,
					Source:  SourceAuto,
				},
			},
			expectedBrowsers: []BrowserSettings{
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  true,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  true,
					Source:  SourceAuto,
				},
			},
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				settings := &Settings{
					Browsers: tt.inputBrowsers,
				}

				normalizedSettings := settings.NormalizeBrowsers()

				assert.Equal(t, tt.expectedBrowsers, normalizedSettings.Browsers)
			},
		)
	}
}

func TestSettings_UpdateWithBrowsers(t *testing.T) {
	for _, tt := range [...]struct {
		name                    string
		inputSettings           *Settings
		inputBrowsers           []Browser
		expectedBrowserSettings []BrowserSettings
	}{
		{
			name: "no changes",
			inputSettings: &Settings{
				Browsers: []BrowserSettings{
					{
						Name:    "Firefox",
						Command: "firefox",
						Hidden:  false,
						Source:  SourceAuto,
					},
					{
						Name:    "Chromium",
						Command: "chromium",
						Hidden:  false,
						Source:  SourceAuto,
					},
				},
			},
			inputBrowsers: []Browser{
				{
					Name:    "Firefox",
					Command: "firefox",
				},
				{
					Name:    "Chromium",
					Command: "chromium",
				},
			},
			expectedBrowserSettings: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
		},
		{
			name: "one browser added appends to the end",
			inputSettings: &Settings{
				Browsers: []BrowserSettings{
					{
						Name:    "Firefox",
						Command: "firefox",
						Hidden:  false,
						Source:  SourceAuto,
					},
					{
						Name:    "Chromium",
						Command: "chromium",
						Hidden:  false,
						Source:  SourceAuto,
					},
				},
			},
			inputBrowsers: []Browser{
				{
					Name:    "Firefox",
					Command: "firefox",
				},
				{
					Name:    "Chromium",
					Command: "chromium",
				},
				{
					Name:    "Brave",
					Command: "brave",
				},
			},
			expectedBrowserSettings: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
		},
		{
			name: "a browser which was auto-added is removed when it's no longer present",
			inputSettings: &Settings{
				Browsers: []BrowserSettings{
					{
						Name:    "Firefox",
						Command: "firefox",
						Hidden:  false,
						Source:  SourceAuto,
					},
					{
						Name:    "Chromium",
						Command: "chromium",
						Hidden:  false,
						Source:  SourceAuto,
					},
					{
						Name:    "Brave",
						Command: "brave",
						Hidden:  false,
						Source:  SourceAuto,
					},
				},
			},
			inputBrowsers: []Browser{
				{
					Name:    "Brave",
					Command: "brave",
				},
				{
					Name:    "Firefox",
					Command: "firefox",
				},
			},
			expectedBrowserSettings: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Brave",
					Command: "brave",
					Hidden:  false,
					Source:  SourceAuto,
				},
			},
		},
		{
			name: "a hidden browser still present on the system stays hidden",
			inputSettings: &Settings{
				Browsers: []BrowserSettings{
					{
						Name:    "Firefox",
						Command: "firefox",
						Hidden:  false,
						Source:  SourceAuto,
					},
					{
						Name:    "Chromium",
						Command: "chromium",
						Hidden:  true,
						Source:  SourceAuto,
					},
				},
			},
			inputBrowsers: []Browser{
				{
					Name:    "Firefox",
					Command: "firefox",
				},
				{
					Name:    "Chromium",
					Command: "chromium",
				},
			},
			expectedBrowserSettings: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Hidden:  false,
					Source:  SourceAuto,
				},
				{
					Name:    "Chromium",
					Command: "chromium",
					Hidden:  true,
					Source:  SourceAuto,
				},
			},
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				settings := tt.inputSettings.UpdateWithBrowsers(tt.inputBrowsers)

				assert.Equal(t, tt.expectedBrowserSettings, settings.Browsers)
			},
		)
	}
}

func TestSettings_UpdateWithBrowsers_PreservesNonBrowserFields(t *testing.T) {
	settings := &Settings{
		LogLevel: "debug",
		Browsers: []BrowserSettings{
			{Name: "Firefox", Command: "firefox", Source: SourceAuto},
		},
		Plugins: []PluginSettings{
			{Path: "/usr/lib/linkquisition/plugins/unwrap.so"},
		},
		Ui: UiSettings{HideKeyboardGuideLabel: true},
	}

	result := settings.UpdateWithBrowsers([]Browser{
		{Name: "Firefox", Command: "firefox"},
		{Name: "Chrome", Command: "chrome"},
	})

	assert.Equal(t, "debug", result.LogLevel)
	assert.Equal(t, 1, len(result.Plugins))
	assert.Equal(t, "/usr/lib/linkquisition/plugins/unwrap.so", result.Plugins[0].Path)
	assert.True(t, result.Ui.HideKeyboardGuideLabel)
}

func TestSettings_GetSelectableBrowsers(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		settings Settings
		expected []Browser
	}{
		{
			name: "returns only visible browsers",
			settings: Settings{
				Browsers: []BrowserSettings{
					{Name: "Firefox", Command: "firefox", Hidden: false},
					{Name: "Chromium", Command: "chromium", Hidden: true},
					{Name: "Brave", Command: "brave", Hidden: false},
				},
			},
			expected: []Browser{
				{Name: "Firefox", Command: "firefox"},
				{Name: "Brave", Command: "brave"},
			},
		},
		{
			name: "returns empty slice when all are hidden",
			settings: Settings{
				Browsers: []BrowserSettings{
					{Name: "Firefox", Command: "firefox", Hidden: true},
					{Name: "Chromium", Command: "chromium", Hidden: true},
				},
			},
			expected: nil,
		},
		{
			name: "returns all when none are hidden",
			settings: Settings{
				Browsers: []BrowserSettings{
					{Name: "Firefox", Command: "firefox", Hidden: false},
					{Name: "Chromium", Command: "chromium", Hidden: false},
				},
			},
			expected: []Browser{
				{Name: "Firefox", Command: "firefox"},
				{Name: "Chromium", Command: "chromium"},
			},
		},
		{
			name:     "returns nil when there are no browsers",
			settings: Settings{Browsers: nil},
			expected: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.GetSelectableBrowsers()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettings_GetMatchingBrowser(t *testing.T) {
	settings := &Settings{
		Browsers: []BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeSite, Value: "www.facebook.com"},
				},
			},
			{
				Name:    "Edge",
				Command: "edge",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeDomain, Value: "office.com"},
				},
			},
		},
	}

	t.Run("returns matching browser for site rule", func(t *testing.T) {
		browser, err := settings.GetMatchingBrowser("https://www.facebook.com/feed")
		assert.NoError(t, err)
		assert.Equal(t, "Firefox", browser.Name)
		assert.Equal(t, "firefox", browser.Command)
	})

	t.Run("returns matching browser for domain rule", func(t *testing.T) {
		browser, err := settings.GetMatchingBrowser("https://outlook.office.com/mail")
		assert.NoError(t, err)
		assert.Equal(t, "Edge", browser.Name)
		assert.Equal(t, "edge", browser.Command)
	})

	t.Run("returns ErrNoMatchFound when no browser matches", func(t *testing.T) {
		browser, err := settings.GetMatchingBrowser("https://github.com/something")
		assert.ErrorIs(t, err, ErrNoMatchFound)
		assert.Nil(t, browser)
	})

	t.Run("returns first match when multiple browsers could match", func(t *testing.T) {
		s := &Settings{
			Browsers: []BrowserSettings{
				{
					Name:    "First",
					Command: "first",
					Matches: []BrowserMatch{{Type: BrowserMatchTypeDomain, Value: "example.com"}},
				},
				{
					Name:    "Second",
					Command: "second",
					Matches: []BrowserMatch{{Type: BrowserMatchTypeDomain, Value: "example.com"}},
				},
			},
		}
		browser, err := s.GetMatchingBrowser("https://www.example.com/page")
		assert.NoError(t, err)
		assert.Equal(t, "First", browser.Name)
	})
}

func TestSettings_AddRuleToBrowser(t *testing.T) {
	t.Run("adds a rule to the matching browser", func(t *testing.T) {
		settings := &Settings{
			Browsers: []BrowserSettings{
				{Name: "Firefox", Command: "firefox"},
				{Name: "Edge", Command: "edge"},
			},
		}

		browser := &Browser{Name: "Firefox", Command: "firefox"}
		settings.AddRuleToBrowser(browser, BrowserMatchTypeSite, "www.example.com")

		assert.Len(t, settings.Browsers[0].Matches, 1)
		assert.Equal(t, BrowserMatchTypeSite, settings.Browsers[0].Matches[0].Type)
		assert.Equal(t, "www.example.com", settings.Browsers[0].Matches[0].Value)
		// Other browser should be untouched
		assert.Empty(t, settings.Browsers[1].Matches)
	})

	t.Run("does nothing when browser command does not match", func(t *testing.T) {
		settings := &Settings{
			Browsers: []BrowserSettings{
				{Name: "Firefox", Command: "firefox"},
			},
		}

		browser := &Browser{Name: "Nonexistent", Command: "nonexistent"}
		settings.AddRuleToBrowser(browser, BrowserMatchTypeDomain, "example.com")

		assert.Empty(t, settings.Browsers[0].Matches)
	})

	t.Run("appends to existing matches", func(t *testing.T) {
		settings := &Settings{
			Browsers: []BrowserSettings{
				{
					Name:    "Firefox",
					Command: "firefox",
					Matches: []BrowserMatch{
						{Type: BrowserMatchTypeSite, Value: "existing.com"},
					},
				},
			},
		}

		browser := &Browser{Name: "Firefox", Command: "firefox"}
		settings.AddRuleToBrowser(browser, BrowserMatchTypeDomain, "new.com")

		assert.Len(t, settings.Browsers[0].Matches, 2)
		assert.Equal(t, "existing.com", settings.Browsers[0].Matches[0].Value)
		assert.Equal(t, "new.com", settings.Browsers[0].Matches[1].Value)
	})
}

func TestMapSettingsLogLevelToSlog(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		input    string
		expected int
	}{
		{name: "debug", input: "debug", expected: -4},
		{name: "info", input: "info", expected: 0},
		{name: "warn", input: "warn", expected: 4},
		{name: "error", input: "error", expected: 8},
		{name: "unknown defaults to info", input: "something", expected: 0},
		{name: "empty defaults to info", input: "", expected: 0},
		{name: "case insensitive DEBUG", input: "DEBUG", expected: -4},
		{name: "case insensitive Warn", input: "Warn", expected: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := MapSettingsLogLevelToSlog(tt.input)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}

func TestBrowserSettings_CompileRegexMatches(t *testing.T) {
	t.Run("valid regex patterns are compiled", func(t *testing.T) {
		browser := BrowserSettings{
			Name:    "Firefox",
			Command: "firefox",
			Matches: []BrowserMatch{
				{Type: BrowserMatchTypeRegex, Value: `.*\.example\.com`},
				{Type: BrowserMatchTypeSite, Value: "www.test.com"},
			},
		}

		browser.CompileRegexMatches()

		// Regex match should work after compilation
		assert.True(t, browser.MatchesUrl("https://sub.example.com/page"))
	})

	t.Run("invalid regex patterns are skipped gracefully", func(t *testing.T) {
		browser := BrowserSettings{
			Name:    "Firefox",
			Command: "firefox",
			Matches: []BrowserMatch{
				{Type: BrowserMatchTypeRegex, Value: `[invalid`},
				{Type: BrowserMatchTypeSite, Value: "www.example.com"},
			},
		}

		// Should not panic
		browser.CompileRegexMatches()

		// Invalid regex should not match
		assert.False(t, browser.MatchesUrl("https://invalid.com"))
		// But site match should still work
		assert.True(t, browser.MatchesUrl("https://www.example.com/page"))
	})
}

func TestSettings_CompileAllRegexMatches(t *testing.T) {
	settings := &Settings{
		Browsers: []BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeRegex, Value: `.*\.mozilla\.org`},
				},
			},
			{
				Name:    "Chrome",
				Command: "chrome",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeRegex, Value: `.*\.google\.com`},
				},
			},
		},
	}

	result := settings.CompileAllRegexMatches()

	// Should return self for chaining
	assert.Same(t, settings, result)

	// Both browsers should have compiled regex matching
	browser, err := settings.GetMatchingBrowser("https://developer.mozilla.org/docs")
	assert.NoError(t, err)
	assert.Equal(t, "Firefox", browser.Name)

	browser, err = settings.GetMatchingBrowser("https://mail.google.com/inbox")
	assert.NoError(t, err)
	assert.Equal(t, "Chrome", browser.Name)
}

func TestBrowserSettings_MatchesUrl_FallbackCompilesRegexOnDemand(t *testing.T) {
	// Simulate a dynamically added regex rule (not pre-compiled)
	browser := BrowserSettings{
		Name:    "Firefox",
		Command: "firefox",
		Matches: []BrowserMatch{
			{Type: BrowserMatchTypeRegex, Value: `https://internal\.corp\..*`},
		},
	}

	// Should still match even without pre-compilation
	assert.True(t, browser.MatchesUrl("https://internal.corp.example.com/app"))
	assert.False(t, browser.MatchesUrl("https://external.example.com"))
}
