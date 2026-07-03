package main_test

import (
	"log/slog"
	"testing"

	"github.com/strobotti/linkquisition/mock"

	"github.com/stretchr/testify/assert"

	"github.com/strobotti/linkquisition"
	. "github.com/strobotti/linkquisition/plugins/unwrap"
)

func TestUnwrap_ModifyUrl_EdgeCases(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name        string
		config      map[string]interface{}
		inputUrl    string
		expectedUrl string
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
			inputUrl:    "https://www.example.com/?url=https%3A%2F%2Fgithub.com",
			expectedUrl: "https://www.example.com/?url=https%3A%2F%2Fgithub.com",
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
			inputUrl:    "https://safelinks.example.com/page?other=value&something=else",
			expectedUrl: "https://safelinks.example.com/page?other=value&something=else",
		},
		{
			name: "no rules configured returns original URL",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules":                       []map[string]interface{}{},
			},
			inputUrl:    "https://www.example.com/something",
			expectedUrl: "https://www.example.com/something",
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
			inputUrl:    "https://second.example.com/link?redirect=https%3A%2F%2Ffinal.example.com",
			expectedUrl: "https://final.example.com",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := Plugin
			provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")
			testedPlugin.Setup(provider, tt.config)

			assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
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
	testedPlugin.Setup(provider, map[string]interface{}{
		"rules": "not-a-list",
	})

	// Plugin should still work — just with no rules, returning URLs unchanged
	result := testedPlugin.ModifyUrl("https://example.com/something")
	assert.Equal(t, "https://example.com/something", result)
}

func TestUnwrap_ModifyUrl(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name            string
		config          map[string]interface{}
		inputUrl        string
		expectedUrl     string
		browserSettings []linkquisition.BrowserSettings
	}{
		{
			name: "Microsoft Teams Defender Safelinks are unwapped",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
			expectedUrl: "https://github.com/Strobotti/linkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwapped if the URL does not match the rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
			expectedUrl: "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwapped if the unwapped URL would not match any rule browsers rules",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
			expectedUrl: "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb",
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
			name: "Microsoft Teams Defender Safelinks are unwapped if the unwapped URL would match any browsers rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
			expectedUrl: "https://github.com/Strobotti/linkquisition",
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
		t.Run(
			tt.name, func(t *testing.T) {
				testedPlugin := Plugin
				provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{Browsers: tt.browserSettings}, "")
				testedPlugin.Setup(provider, tt.config)

				assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
			},
		)
	}
}
