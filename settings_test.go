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
			name: "site does not match even the domain is the same",
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
