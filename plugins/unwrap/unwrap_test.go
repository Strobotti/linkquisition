package main_test

import (
	"log/slog"
	"testing"

	"github.com/strobotti/linkquisition/mock"

	"github.com/stretchr/testify/assert"

	"github.com/strobotti/linkquisition"
	. "github.com/strobotti/linkquisition/plugins/unwrap"
)

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
				provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{Browsers: tt.browserSettings})
				testedPlugin.Setup(provider, tt.config)

				assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
			},
		)
	}
}
