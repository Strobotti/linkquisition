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

func TestBrowserSettings_MatchesUrl_CaseInsensitive(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		settings BrowserSettings
		url      string
		expected bool
	}{
		{
			name: "site match is case-insensitive (rule uppercase, URL lowercase)",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeSite, Value: "WWW.EXAMPLE.COM"},
				},
			},
			url:      "https://www.example.com/page",
			expected: true,
		},
		{
			name: "site match is case-insensitive (rule mixed case, URL lowercase)",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeSite, Value: "Www.Example.Com"},
				},
			},
			url:      "https://www.example.com/page",
			expected: true,
		},
		{
			name: "domain match is case-insensitive (rule uppercase)",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeDomain, Value: "EXAMPLE.COM"},
				},
			},
			url:      "https://www.example.com/page",
			expected: true,
		},
		{
			name: "domain match is case-insensitive (URL uppercase)",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeDomain, Value: "example.com"},
				},
			},
			url:      "https://WWW.EXAMPLE.COM/page",
			expected: true,
		},
		{
			name: "site match is case-insensitive (URL uppercase)",
			settings: BrowserSettings{
				Name:    "Firefox",
				Command: "firefox",
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeSite, Value: "www.example.com"},
				},
			},
			url:      "https://WWW.EXAMPLE.COM/page",
			expected: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.settings.MatchesUrl(tt.url))
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
