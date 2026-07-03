package main_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/mock"
	. "github.com/strobotti/linkquisition/plugins/sanitize"
)

func TestSanitize_Setup_Defaults(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{})

	testedPlugin := Plugin
	testedPlugin.Setup(provider, map[string]interface{}{})

	// With defaults, tracking params should be stripped
	result := testedPlugin.ModifyUrl("https://example.com/page?utm_source=twitter&utm_medium=social&title=hello")
	assert.Equal(t, "https://example.com/page?title=hello", result)
}

func TestSanitize_Setup_InvalidConfig(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{})

	testedPlugin := Plugin
	// Pass config with wrong type — triggers mapstructure decode error
	testedPlugin.Setup(provider, map[string]interface{}{
		"stripDefaults": "not-a-bool",
	})

	// Plugin should still work with defaults (stripDefaults=true)
	result := testedPlugin.ModifyUrl("https://example.com/?utm_source=test")
	// After invalid config, defaults should apply — but since decode failed, stripDefaults stays true
	assert.Equal(t, "https://example.com/", result)
}

func TestSanitize_ModifyUrl(t *testing.T) {
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
			name:        "URL without query params is unchanged",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page",
			expectedUrl: "https://example.com/page",
		},
		{
			name:        "URL with no tracking params is unchanged",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?id=123&name=test",
			expectedUrl: "https://example.com/page?id=123&name=test",
		},
		{
			name:        "UTM params are stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?utm_source=twitter&utm_medium=social&id=123",
			expectedUrl: "https://example.com/page?id=123",
		},
		{
			name:        "fbclid is stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?fbclid=abc123&id=456",
			expectedUrl: "https://example.com/page?id=456",
		},
		{
			name:        "gclid is stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?gclid=xyz789&product=widget",
			expectedUrl: "https://example.com/page?product=widget",
		},
		{
			name:        "msclkid is stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?msclkid=abc&q=search",
			expectedUrl: "https://example.com/page?q=search",
		},
		{
			name:        "all tracking params stripped leaves clean URL",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?utm_source=x&utm_medium=y&utm_campaign=z",
			expectedUrl: "https://example.com/page",
		},
		{
			name: "stripDefaults=false disables default list",
			config: map[string]interface{}{
				"stripDefaults": false,
			},
			inputUrl:    "https://example.com/page?utm_source=twitter&id=123",
			expectedUrl: "https://example.com/page?utm_source=twitter&id=123",
		},
		{
			name: "extraParams are stripped",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraParams":   []interface{}{"tracking_id", "ref"},
			},
			inputUrl:    "https://example.com/page?tracking_id=abc&ref=homepage&id=123",
			expectedUrl: "https://example.com/page?id=123",
		},
		{
			name: "extraPatterns work with regex",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraPatterns": []interface{}{"^_ga", "^itm_"},
			},
			inputUrl:    "https://example.com/page?_ga=123&_gac=456&itm_source=nav&id=789",
			expectedUrl: "https://example.com/page?id=789",
		},
		{
			name: "onlyMatchingUrls limits sanitization to matching URLs",
			config: map[string]interface{}{
				"onlyMatchingUrls": "^https://example\\.com",
			},
			inputUrl:    "https://other.com/page?utm_source=twitter&id=123",
			expectedUrl: "https://other.com/page?utm_source=twitter&id=123",
		},
		{
			name: "onlyMatchingUrls allows sanitization for matching URLs",
			config: map[string]interface{}{
				"onlyMatchingUrls": "^https://example\\.com",
			},
			inputUrl:    "https://example.com/page?utm_source=twitter&id=123",
			expectedUrl: "https://example.com/page?id=123",
		},
		{
			name: "invalid onlyMatchingUrls regex is handled gracefully",
			config: map[string]interface{}{
				"onlyMatchingUrls": "[invalid",
			},
			inputUrl:    "https://example.com/page?utm_source=twitter&id=123",
			expectedUrl: "https://example.com/page?id=123",
		},
		{
			name: "invalid extraPatterns regex entries are skipped",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraPatterns": []interface{}{"[invalid", "^valid_"},
			},
			inputUrl:    "https://example.com/page?valid_tracking=1&other=2",
			expectedUrl: "https://example.com/page?other=2",
		},
		{
			name:        "URL with fragment is preserved",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?utm_source=x&id=1#section",
			expectedUrl: "https://example.com/page?id=1#section",
		},
		{
			name:        "malformed URL is returned unchanged",
			config:      map[string]interface{}{},
			inputUrl:    "://not-a-valid-url",
			expectedUrl: "://not-a-valid-url",
		},
		{
			name: "extraParams and defaults work together",
			config: map[string]interface{}{
				"extraParams": []interface{}{"custom_track"},
			},
			inputUrl:    "https://example.com/page?utm_source=x&custom_track=y&id=1",
			expectedUrl: "https://example.com/page?id=1",
		},
		{
			name:        "HubSpot params are stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?_hsenc=abc&_hsmi=123&title=hello",
			expectedUrl: "https://example.com/page?title=hello",
		},
		{
			name:        "Mailchimp params are stripped by default",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page?mc_cid=abc&mc_eid=123&article=news",
			expectedUrl: "https://example.com/page?article=news",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := Plugin
			provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{})
			testedPlugin.Setup(provider, tt.config)

			assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
		})
	}
}
