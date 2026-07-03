package main_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// isWarningPage checks if a URL points to a defang warning page file
func isWarningPage(url string) bool {
	return strings.HasPrefix(url, "file://") && strings.Contains(url, "linkquisition-defang-")
}

func TestDefang_Setup_Defaults(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{})

	// With no cached lists, should not block anything
	result := testedPlugin.ModifyUrl("https://example.com/page")
	assert.Equal(t, "https://example.com/page", result)
}

func TestDefang_Setup_InvalidConfig(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"updateInterval": "not-a-duration",
		"action":         123, // wrong type
	})

	// Should still work with defaults
	result := testedPlugin.ModifyUrl("https://example.com/page")
	assert.Equal(t, "https://example.com/page", result)
}

func TestDefang_Setup_InvalidAction(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"action": "unknown_action",
	})

	// Should fall back to "block" action
	result := testedPlugin.ModifyUrl("https://example.com/page")
	assert.Equal(t, "https://example.com/page", result)
}

func TestDefang_ModifyUrl_WithCachedBlocklist(t *testing.T) {
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
		name        string
		config      map[string]interface{}
		inputUrl    string
		expectedUrl string
	}{
		{
			name:        "blocked domain returns warning page",
			config:      map[string]interface{}{},
			inputUrl:    "https://malware.example.com/payload",
			expectedUrl: "WARNING_PAGE",
		},
		{
			name:        "another blocked domain returns warning page",
			config:      map[string]interface{}{},
			inputUrl:    "https://phishing.example.net/login",
			expectedUrl: "WARNING_PAGE",
		},
		{
			name:        "127.0.0.1 entries are also blocked",
			config:      map[string]interface{}{},
			inputUrl:    "https://ads.tracker.io/track?id=123",
			expectedUrl: "WARNING_PAGE",
		},
		{
			name:        "safe domain is not blocked",
			config:      map[string]interface{}{},
			inputUrl:    "https://safe.example.com/page",
			expectedUrl: "https://safe.example.com/page",
		},
		{
			name:        "localhost is not blocked",
			config:      map[string]interface{}{},
			inputUrl:    "http://localhost:8080/api",
			expectedUrl: "http://localhost:8080/api",
		},
		{
			name:        "action=log returns original URL",
			config:      map[string]interface{}{"action": "log"},
			inputUrl:    "https://malware.example.com/payload",
			expectedUrl: "https://malware.example.com/payload",
		},
		{
			name:        "action=warn shows warning page with proceed option",
			config:      map[string]interface{}{"action": "warn"},
			inputUrl:    "https://malware.example.com/payload",
			expectedUrl: "WARNING_PAGE_WITH_PROCEED",
		},
		{
			name:        "subdomain of blocked domain is also blocked",
			config:      map[string]interface{}{},
			inputUrl:    "https://sub.malware.example.com/page",
			expectedUrl: "WARNING_PAGE",
		},
		{
			name:        "parent domain of blocked domain is NOT blocked",
			config:      map[string]interface{}{},
			inputUrl:    "https://example.com/page",
			expectedUrl: "https://example.com/page",
		},
		{
			name:        "URL with port on blocked domain is blocked",
			config:      map[string]interface{}{},
			inputUrl:    "https://malware.example.com:8443/path",
			expectedUrl: "WARNING_PAGE",
		},
		{
			name:        "invalid URL is returned unchanged",
			config:      map[string]interface{}{},
			inputUrl:    "://invalid",
			expectedUrl: "://invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := newPlugin()
			// Use long interval so no background fetch is triggered
			cfg := map[string]interface{}{
				"updateInterval": "87600h",
				"sources":        []interface{}{},
			}
			for k, v := range tt.config {
				cfg[k] = v
			}
			testedPlugin.Setup(provider, cfg)

			result := testedPlugin.ModifyUrl(tt.inputUrl)
			switch tt.expectedUrl {
			case "WARNING_PAGE":
				assert.True(t, isWarningPage(result), "expected warning page, got: %s", result)
				// Verify the file exists and does NOT contain "Open anyway"
				content, err := os.ReadFile(strings.TrimPrefix(result, "file://"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "blocklist")
				assert.NotContains(t, string(content), "Open anyway")
			case "WARNING_PAGE_WITH_PROCEED":
				assert.True(t, isWarningPage(result), "expected warning page, got: %s", result)
				// Verify the file exists and DOES contain "Open anyway"
				content, err := os.ReadFile(strings.TrimPrefix(result, "file://"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "Open anyway")
			default:
				assert.Equal(t, tt.expectedUrl, result)
			}
		})
	}
}

func TestDefang_BackgroundFetch(t *testing.T) {
	// Set up a test HTTP server serving a hosts file
	hostsContent := `0.0.0.0 fetched-malware.test
0.0.0.0 fetched-phishing.test
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(hostsContent))
	}))
	defer server.Close()

	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{server.URL},
		"updateInterval": "1ms", // Force immediate update
	})

	// Wait for the background fetch to complete via Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testedPlugin.Shutdown(ctx)

	// Now the fetched domains should be blocked
	result1 := testedPlugin.ModifyUrl("https://fetched-malware.test/page")
	assert.True(t, isWarningPage(result1), "expected warning page, got: %s", result1)
	result2 := testedPlugin.ModifyUrl("https://fetched-phishing.test/login")
	assert.True(t, isWarningPage(result2), "expected warning page, got: %s", result2)
	assert.Equal(t, "https://safe.test/page", testedPlugin.ModifyUrl("https://safe.test/page"))

	// Verify the cache file was written
	_, err := os.Stat(filepath.Join(tmpDir, "defang", "source_0.txt"))
	assert.NoError(t, err)
}

func TestDefang_BackgroundFetch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{server.URL},
		"updateInterval": "1ms",
	})

	// Wait for fetch to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testedPlugin.Shutdown(ctx)

	// Nothing should be blocked since the fetch failed
	assert.Equal(t, "https://anything.test/page", testedPlugin.ModifyUrl("https://anything.test/page"))
}

func TestDefang_Shutdown_NoUpdate(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{},
		"updateInterval": "87600h",
	})

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
	testedPlugin.Setup(provider, map[string]interface{}{
		"updateInterval": "87600h",
		"sources":        []interface{}{},
	})

	// Domain lookup should be case-insensitive
	result1 := testedPlugin.ModifyUrl("https://MALWARE.EXAMPLE.COM/page")
	assert.True(t, isWarningPage(result1), "expected warning page, got: %s", result1)
	result2 := testedPlugin.ModifyUrl("https://malware.example.com/page")
	assert.True(t, isWarningPage(result2), "expected warning page, got: %s", result2)
}

func TestDefang_CustomSources(t *testing.T) {
	logger := newTestLogger()
	tmpDir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, tmpDir)

	testedPlugin := newPlugin()
	testedPlugin.Setup(provider, map[string]interface{}{
		"sources":        []interface{}{"https://custom.source/hosts"},
		"updateInterval": "87600h", // don't actually fetch
	})

	// Should not crash with unreachable sources (just no domains loaded)
	assert.Equal(t, "https://example.com/page", testedPlugin.ModifyUrl("https://example.com/page"))
}
