package main_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/strobotti/linkquisition/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
	. "github.com/strobotti/linkquisition/plugins/unwrap"
)

func TestUnwrap_ProcessURL_EdgeCases(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name        string
		config      map[string]interface{}
		inputURL    string
		expectedURL string
	}{
		{
			name: "empty match rule is skipped gracefully",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://www.example.com/?url=https%3A%2F%2Fgithub.com",
			expectedURL: "https://www.example.com/?url=https%3A%2F%2Fgithub.com",
		},
		{
			name: "matching URL but missing query parameter returns original URL",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://safelinks\\.example\\.com",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://safelinks.example.com/page?other=value&something=else",
			expectedURL: "https://safelinks.example.com/page?other=value&something=else",
		},
		{
			name: "no rules configured returns original URL",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules":                       []map[string]interface{}{},
			},
			inputURL:    "https://www.example.com/something",
			expectedURL: "https://www.example.com/something",
		},
		{
			name: "multiple rules, second one matches",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://first\\.example\\.com",
						"parameter": "target",
					},
					{
						"match":     "^https://second\\.example\\.com",
						"parameter": "redirect",
					},
				},
			},
			inputURL:    "https://second.example.com/link?redirect=https%3A%2F%2Ffinal.example.com",
			expectedURL: "https://final.example.com",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := Plugin
			provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")
			err := testedPlugin.Setup(provider, tt.config)
			require.NoError(t, err)

			result := testedPlugin.ProcessURL(context.Background(), tt.inputURL)
			assert.Equal(t, linkquisition.ActionContinue, result.Action)
			assert.True(t, result.ContinueChain)
			assert.Equal(t, tt.expectedURL, result.URL)
		})
	}
}

func TestUnwrap_Setup_InvalidConfig(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	// Pass config with wrong type for "rules" field — triggers mapstructure decode error
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"rules": "not-a-list",
	})
	assert.Error(t, err)
}

func TestUnwrap_ProcessURL(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name            string
		config          map[string]interface{}
		inputURL        string
		expectedURL     string
		browserSettings []linkquisition.BrowserSettings
	}{
		{
			name: "Microsoft Teams Defender Safelinks are unwrapped",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
			expectedURL: "https://github.com/Strobotti/linkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwrapped if the URL does not match the rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
			expectedURL: "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwrapped if the unwrapped URL would not match any browser rules",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
			expectedURL: "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
			browserSettings: []linkquisition.BrowserSettings{
				{
					Name: "Test Browser",
					Matches: []linkquisition.BrowserMatch{
						{
							Type:  "domain",
							Value: "example.com",
						},
					},
				},
			},
		},
		{
			name: "Microsoft Teams Defender Safelinks are unwrapped if the unwrapped URL would match a browser rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputURL:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
			expectedURL: "https://github.com/Strobotti/linkquisition",
			browserSettings: []linkquisition.BrowserSettings{
				{
					Name: "Test Browser",
					Matches: []linkquisition.BrowserMatch{
						{
							Type:  "domain",
							Value: "github.com",
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := Plugin
			provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{Browsers: tt.browserSettings}, "")
			err := testedPlugin.Setup(provider, tt.config)
			require.NoError(t, err)

			result := testedPlugin.ProcessURL(context.Background(), tt.inputURL)
			assert.Equal(t, linkquisition.ActionContinue, result.Action)
			assert.Equal(t, tt.expectedURL, result.URL)
		})
	}
}

func TestUnwrap_Metadata(t *testing.T) {
	testedPlugin := Plugin
	meta := testedPlugin.Metadata()

	assert.Equal(t, "Unwrap", meta.Name)
	assert.NotEmpty(t, meta.Description)
	assert.Len(t, meta.Settings, 2)
}
