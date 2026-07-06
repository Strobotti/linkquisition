package main_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	err := testedPlugin.Setup(provider, map[string]interface{}{})
	require.NoError(t, err)

	// With defaults, tracking params should be stripped
	result := testedPlugin.ProcessURL(context.Background(), "https://example.com/page?utm_source=twitter&utm_medium=social&title=hello")
	assert.Equal(t, linkquisition.ActionContinue, result.Action)
	assert.Equal(t, "https://example.com/page?title=hello", result.URL)
}

func TestSanitize_Setup_InvalidRegex(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"onlyMatchingUrls": "[invalid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid onlyMatchingUrls regex")
}

func TestSanitize_ProcessURL(t *testing.T) {
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
			name:        "URL without query params is unchanged",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page",
			expectedURL: "https://example.com/page",
		},
		{
			name:        "URL with no tracking params is unchanged",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?id=123&name=test",
			expectedURL: "https://example.com/page?id=123&name=test",
		},
		{
			name:        "UTM params are stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?utm_source=twitter&utm_medium=social&id=123",
			expectedURL: "https://example.com/page?id=123",
		},
		{
			name:        "fbclid is stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?fbclid=abc123&id=456",
			expectedURL: "https://example.com/page?id=456",
		},
		{
			name:        "gclid is stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?gclid=xyz789&product=widget",
			expectedURL: "https://example.com/page?product=widget",
		},
		{
			name:        "msclkid is stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?msclkid=abc&q=search",
			expectedURL: "https://example.com/page?q=search",
		},
		{
			name:        "all tracking params stripped leaves clean URL",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?utm_source=x&utm_medium=y&utm_campaign=z",
			expectedURL: "https://example.com/page",
		},
		{
			name: "stripDefaults=false disables default list",
			config: map[string]interface{}{
				"stripDefaults": false,
			},
			inputURL:    "https://example.com/page?utm_source=twitter&id=123",
			expectedURL: "https://example.com/page?utm_source=twitter&id=123",
		},
		{
			name: "extraParams are stripped",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraParams":   []interface{}{"tracking_id", "ref"},
			},
			inputURL:    "https://example.com/page?tracking_id=abc&ref=homepage&id=123",
			expectedURL: "https://example.com/page?id=123",
		},
		{
			name: "extraPatterns work with regex",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraPatterns": []interface{}{"^_ga", "^itm_"},
			},
			inputURL:    "https://example.com/page?_ga=123&_gac=456&itm_source=nav&id=789",
			expectedURL: "https://example.com/page?id=789",
		},
		{
			name: "onlyMatchingUrls limits sanitization to matching URLs",
			config: map[string]interface{}{
				"onlyMatchingUrls": "^https://example\\.com",
			},
			inputURL:    "https://other.com/page?utm_source=twitter&id=123",
			expectedURL: "https://other.com/page?utm_source=twitter&id=123",
		},
		{
			name: "onlyMatchingUrls allows sanitization for matching URLs",
			config: map[string]interface{}{
				"onlyMatchingUrls": "^https://example\\.com",
			},
			inputURL:    "https://example.com/page?utm_source=twitter&id=123",
			expectedURL: "https://example.com/page?id=123",
		},
		{
			name: "invalid extraPatterns regex entries are skipped",
			config: map[string]interface{}{
				"stripDefaults": false,
				"extraPatterns": []interface{}{"[invalid", "^valid_"},
			},
			inputURL:    "https://example.com/page?valid_tracking=1&other=2",
			expectedURL: "https://example.com/page?other=2",
		},
		{
			name:        "URL with fragment is preserved",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?utm_source=x&id=1#section",
			expectedURL: "https://example.com/page?id=1#section",
		},
		{
			name:        "malformed URL is returned unchanged",
			config:      map[string]interface{}{},
			inputURL:    "://not-a-valid-url",
			expectedURL: "://not-a-valid-url",
		},
		{
			name: "extraParams and defaults work together",
			config: map[string]interface{}{
				"extraParams": []interface{}{"custom_track"},
			},
			inputURL:    "https://example.com/page?utm_source=x&custom_track=y&id=1",
			expectedURL: "https://example.com/page?id=1",
		},
		{
			name:        "HubSpot params are stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?_hsenc=abc&_hsmi=123&title=hello",
			expectedURL: "https://example.com/page?title=hello",
		},
		{
			name:        "Mailchimp params are stripped by default",
			config:      map[string]interface{}{},
			inputURL:    "https://example.com/page?mc_cid=abc&mc_eid=123&article=news",
			expectedURL: "https://example.com/page?article=news",
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

func TestSanitize_Metadata(t *testing.T) {
	testedPlugin := Plugin
	meta := testedPlugin.Metadata()

	assert.Equal(t, "Sanitize", meta.Name)
	assert.NotEmpty(t, meta.Description)
	assert.Len(t, meta.Settings, 4)
}
