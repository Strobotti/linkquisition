package main_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/mock"
	. "github.com/strobotti/linkquisition/plugins/defang"
)

func newTestLogger() *slog.Logger {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	return slog.New(slog.NewTextHandler(mockIoWriter, nil))
}

// newPlugin creates a fresh plugin instance for testing (avoids sharing mutex state between tests)
func newPlugin() linkquisition.Plugin {
	return NewForTesting()
}

func TestDefang_Setup_Defaults(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://dummy.test/hosts"},
		"updateInterval": "87600h",
	})
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		testedPlugin.Shutdown(ctx)
	}()

	// With no cached lists, should not block anything
	result := testedPlugin.ProcessURL(context.Background(), "https://example.com/page")
	assert.Equal(t, linkquisition.ActionContinue, result.Action)
	assert.Equal(t, "https://example.com/page", result.URL)
}

func TestDefang_Setup_InvalidDuration(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://dummy.test/hosts"},
		"updateInterval": "not-a-duration",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid updateInterval")
}

func TestDefang_Setup_InvalidAction(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://dummy.test/hosts"},
		"updateInterval": "87600h",
		"action":         "unknown_action",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestDefang_ProcessURL_WithCachedBlocklist(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()

	// Create a cached blocklist
	cacheDir := filepath.Join(tmpDir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))

	hostsContent := `# Comment line
0.0.0.0 malware.example.com
0.0.0.0 phishing.example.net
127.0.0.1 ads.tracker.io
0.0.0.0 localhost
`
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "source_0.txt"), []byte(hostsContent), 0600))

	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	for _, tt := range []struct {
		name           string
		config         map[string]interface{}
		inputURL       string
		expectedAction linkquisition.PluginAction
		expectedURL    string
		hasMessage     bool
	}{
		{
			name:           "blocked domain returns ActionBlock",
			config:         map[string]interface{}{},
			inputURL:       "https://malware.example.com/payload",
			expectedAction: linkquisition.ActionBlock,
			expectedURL:    "https://malware.example.com/payload",
			hasMessage:     true,
		},
		{
			name:           "another blocked domain returns ActionBlock",
			config:         map[string]interface{}{},
			inputURL:       "https://phishing.example.net/login",
			expectedAction: linkquisition.ActionBlock,
			expectedURL:    "https://phishing.example.net/login",
			hasMessage:     true,
		},
		{
			name:           "127.0.0.1 entries are also blocked",
			config:         map[string]interface{}{},
			inputURL:       "https://ads.tracker.io/track?id=123",
			expectedAction: linkquisition.ActionBlock,
			expectedURL:    "https://ads.tracker.io/track?id=123",
			hasMessage:     true,
		},
		{
			name:           "safe domain is not blocked",
			config:         map[string]interface{}{},
			inputURL:       "https://safe.example.com/page",
			expectedAction: linkquisition.ActionContinue,
			expectedURL:    "https://safe.example.com/page",
		},
		{
			name:           "localhost is not blocked",
			config:         map[string]interface{}{},
			inputURL:       "http://localhost:8080/api",
			expectedAction: linkquisition.ActionContinue,
			expectedURL:    "http://localhost:8080/api",
		},
		{
			name:           "action=log returns ActionContinue",
			config:         map[string]interface{}{"action": "log"},
			inputURL:       "https://malware.example.com/payload",
			expectedAction: linkquisition.ActionContinue,
			expectedURL:    "https://malware.example.com/payload",
		},
		{
			name:           "action=warn returns ActionWarn",
			config:         map[string]interface{}{"action": "warn"},
			inputURL:       "https://malware.example.com/payload",
			expectedAction: linkquisition.ActionWarn,
			expectedURL:    "https://malware.example.com/payload",
			hasMessage:     true,
		},
		{
			name:           "subdomain of blocked domain is also blocked",
			config:         map[string]interface{}{},
			inputURL:       "https://sub.malware.example.com/page",
			expectedAction: linkquisition.ActionBlock,
			expectedURL:    "https://sub.malware.example.com/page",
			hasMessage:     true,
		},
		{
			name:           "parent domain of blocked domain is NOT blocked",
			config:         map[string]interface{}{},
			inputURL:       "https://example.com/page",
			expectedAction: linkquisition.ActionContinue,
			expectedURL:    "https://example.com/page",
		},
		{
			name:           "URL with port on blocked domain is blocked",
			config:         map[string]interface{}{},
			inputURL:       "https://malware.example.com:8443/path",
			expectedAction: linkquisition.ActionBlock,
			expectedURL:    "https://malware.example.com:8443/path",
			hasMessage:     true,
		},
		{
			name:           "invalid URL is returned unchanged",
			config:         map[string]interface{}{},
			inputURL:       "://invalid",
			expectedAction: linkquisition.ActionContinue,
			expectedURL:    "://invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := newPlugin()
			// Use a single custom source matching our pre-created cache file
			// and a long interval so no background fetch is triggered
			cfg := map[string]interface{}{
				"updateInterval": "87600h",
				"sources":        []interface{}{"https://dummy.test/hosts"},
			}
			for k, v := range tt.config {
				cfg[k] = v
			}
			err := testedPlugin.Setup(provider, cfg)
			require.NoError(t, err)

			result := testedPlugin.ProcessURL(context.Background(), tt.inputURL)
			assert.Equal(t, tt.expectedAction, result.Action)
			assert.Equal(t, tt.expectedURL, result.URL)
			if tt.hasMessage {
				assert.NotEmpty(t, result.Message)
				assert.Contains(t, result.Message, "blocklist")
			}
		})
	}
}

func TestDefang_BackgroundFetch(t *testing.T) {
	// Set up a test HTTP server serving a hosts file
	hostsContent := `0.0.0.0 fetched-malware.test
0.0.0.0 fetched-phishing.test
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(hostsContent))
	}))
	defer server.Close()

	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{server.URL},
		"updateInterval": "1ms", // Force immediate update
	})
	require.NoError(t, err)

	// Wait for the background fetch to complete via Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testedPlugin.Shutdown(ctx)

	// Now the fetched domains should be blocked
	result1 := testedPlugin.ProcessURL(context.Background(), "https://fetched-malware.test/page")
	assert.Equal(t, linkquisition.ActionBlock, result1.Action)
	result2 := testedPlugin.ProcessURL(context.Background(), "https://fetched-phishing.test/login")
	assert.Equal(t, linkquisition.ActionBlock, result2.Action)

	safeResult := testedPlugin.ProcessURL(context.Background(), "https://safe.test/page")
	assert.Equal(t, linkquisition.ActionContinue, safeResult.Action)
	assert.Equal(t, "https://safe.test/page", safeResult.URL)

	// Verify the cache file was written
	_, statErr := os.Stat(filepath.Join(tmpDir, "defang", "source_0.txt"))
	assert.NoError(t, statErr)
}

func TestDefang_BackgroundFetch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{server.URL},
		"updateInterval": "1ms",
	})
	require.NoError(t, err)

	// Wait for fetch to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testedPlugin.Shutdown(ctx)

	// Nothing should be blocked since the fetch failed
	result := testedPlugin.ProcessURL(context.Background(), "https://anything.test/page")
	assert.Equal(t, linkquisition.ActionContinue, result.Action)
	assert.Equal(t, "https://anything.test/page", result.URL)
}

func TestDefang_Shutdown_NoUpdate(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()

	// Pre-create cache so no update is triggered
	cacheDir := filepath.Join(tmpDir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "source_0.txt"), []byte(""), 0600))

	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://dummy.test/hosts"},
		"updateInterval": "87600h",
	})
	require.NoError(t, err)

	// Shutdown should return immediately when no update is in progress
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	testedPlugin.Shutdown(ctx)
	// If we get here without blocking, the test passes
}

func TestDefang_CaseInsensitive(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()

	cacheDir := filepath.Join(tmpDir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))

	hostsContent := `0.0.0.0 Malware.Example.COM
`
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "source_0.txt"), []byte(hostsContent), 0600))

	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"updateInterval": "87600h",
		"sources":        []interface{}{"https://dummy.test/hosts"},
	})
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		testedPlugin.Shutdown(ctx)
	}()

	// Domain lookup should be case-insensitive
	result1 := testedPlugin.ProcessURL(context.Background(), "https://MALWARE.EXAMPLE.COM/page")
	assert.Equal(t, linkquisition.ActionBlock, result1.Action)
	result2 := testedPlugin.ProcessURL(context.Background(), "https://malware.example.com/page")
	assert.Equal(t, linkquisition.ActionBlock, result2.Action)
}

func TestDefang_CustomSources(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://custom.source/hosts"},
		"updateInterval": "87600h", // don't actually fetch
	})
	require.NoError(t, err)

	// Should not crash with unreachable sources (just no domains loaded)
	result := testedPlugin.ProcessURL(context.Background(), "https://example.com/page")
	assert.Equal(t, linkquisition.ActionContinue, result.Action)
	assert.Equal(t, "https://example.com/page", result.URL)

	// Wait for background update goroutine to finish before TempDir cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	testedPlugin.Shutdown(ctx)
}

func TestDefang_Metadata(t *testing.T) {
	testedPlugin := newPlugin()
	meta := testedPlugin.Metadata()

	assert.Equal(t, "Defang", meta.Name)
	assert.NotEmpty(t, meta.Description)
	assert.NotEmpty(t, meta.Author)
	assert.NotEmpty(t, meta.Version)
	assert.Len(t, meta.Settings, 3)

	// Verify settings descriptors
	assert.Equal(t, "sources", meta.Settings[0].Key)
	assert.Equal(t, linkquisition.SettingTypeStringList, meta.Settings[0].Type)
	assert.Equal(t, "updateInterval", meta.Settings[1].Key)
	assert.Equal(t, linkquisition.SettingTypeDuration, meta.Settings[1].Type)
	assert.Equal(t, "action", meta.Settings[2].Key)
	assert.Equal(t, linkquisition.SettingTypeChoice, meta.Settings[2].Type)
	assert.Equal(t, []string{"block", "warn", "log"}, meta.Settings[2].Options)
}
