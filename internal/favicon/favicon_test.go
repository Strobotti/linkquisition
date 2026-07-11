package favicon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_FetchDirect(t *testing.T) {
	faviconData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(faviconData)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyDirect, "")
	data, err := fetcher.Fetch(context.Background(), server.URL+"/some/page")

	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
}

func TestFetcher_FetchDirect_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyDirect, "")
	_, err := fetcher.Fetch(context.Background(), server.URL+"/page")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestFetcher_FetchParsed(t *testing.T) {
	faviconData := []byte{0x00, 0x00, 0x01, 0x00} // ICO magic bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `<html><head><link rel="icon" href="/assets/icon.png"></head></html>`)
		case "/assets/icon.png":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(faviconData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyParsed, "")
	data, err := fetcher.Fetch(context.Background(), server.URL+"/page")

	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
}

func TestFetcher_FetchParsed_FallbackToDirect(t *testing.T) {
	faviconData := []byte{0x89, 0x50, 0x4E, 0x47}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// HTML without any link icon tag
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `<html><head><title>No Icon</title></head></html>`)
		case "/favicon.ico":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(faviconData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyParsed, "")
	data, err := fetcher.Fetch(context.Background(), server.URL+"/page")

	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
}

func TestFetcher_FetchGoogle(t *testing.T) {
	faviconData := []byte{0x89, 0x50, 0x4E, 0x47}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(faviconData)
	}))
	defer server.Close()

	// Override the google URL to point to our test server
	fetcher := &Fetcher{
		strategy: StrategyGoogle,
		cacheDir: "",
		client:   &http.Client{Timeout: fetchTimeout},
	}

	// We can't easily test the actual Google URL, so test the download mechanism
	data, err := fetcher.download(context.Background(), server.URL+"/favicon")

	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
}

func TestFetcher_Cache(t *testing.T) {
	cacheDir := t.TempDir()
	faviconData := []byte{0x89, 0x50, 0x4E, 0x47}
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(faviconData)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyDirect, cacheDir)

	// First fetch should hit the server
	data, err := fetcher.Fetch(context.Background(), server.URL+"/page")
	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
	assert.Equal(t, 1, callCount)

	// Second fetch should come from cache
	data, err = fetcher.Fetch(context.Background(), server.URL+"/other-page")
	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
	assert.Equal(t, 1, callCount) // no additional server call
}

func TestFetcher_Cache_Expired(t *testing.T) {
	cacheDir := t.TempDir()
	faviconData := []byte{0x89, 0x50, 0x4E, 0x47}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(faviconData)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyDirect, cacheDir)

	// Write an expired cache entry manually
	host := "127.0.0.1"
	cachePath := filepath.Join(cacheDir, cacheKey(host))
	require.NoError(t, os.MkdirAll(cacheDir, cacheDirPerms))
	require.NoError(t, os.WriteFile(cachePath, []byte("old data"), cacheFilePerms))

	// Set modification time to the past
	expiredTime := time.Now().Add(-cacheTTL - time.Hour)
	require.NoError(t, os.Chtimes(cachePath, expiredTime, expiredTime))

	// Should fetch fresh data, not use the expired cache
	data, err := fetcher.Fetch(context.Background(), server.URL+"/page")
	require.NoError(t, err)
	assert.Equal(t, faviconData, data)
}

func TestFetcher_InvalidURL(t *testing.T) {
	fetcher := NewFetcher(StrategyDirect, "")

	_, err := fetcher.Fetch(context.Background(), "://invalid")
	assert.Error(t, err)
}

func TestFetcher_EmptyHost(t *testing.T) {
	fetcher := NewFetcher(StrategyDirect, "")

	_, err := fetcher.Fetch(context.Background(), "file:///local/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no host")
}

func TestFetcher_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := NewFetcher(StrategyDirect, "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := fetcher.Fetch(ctx, server.URL+"/page")
	assert.Error(t, err)
}

func TestResolveIconURL(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		href     string
		baseURL  string
		expected string
	}{
		{
			name:     "absolute https URL",
			href:     "https://cdn.example.com/icon.png",
			baseURL:  "https://example.com/",
			expected: "https://cdn.example.com/icon.png",
		},
		{
			name:     "absolute http URL",
			href:     "http://cdn.example.com/icon.png",
			baseURL:  "https://example.com/",
			expected: "http://cdn.example.com/icon.png",
		},
		{
			name:     "protocol-relative URL",
			href:     "//cdn.example.com/icon.png",
			baseURL:  "https://example.com/",
			expected: "https://cdn.example.com/icon.png",
		},
		{
			name:     "relative path",
			href:     "/assets/favicon.ico",
			baseURL:  "https://example.com/",
			expected: "https://example.com/assets/favicon.ico",
		},
		{
			name:     "relative path without leading slash",
			href:     "images/icon.png",
			baseURL:  "https://example.com/pages/",
			expected: "https://example.com/pages/images/icon.png",
		},
		{
			name:     "empty href",
			href:     "",
			baseURL:  "https://example.com/",
			expected: "",
		},
		{
			name:     "whitespace-only href",
			href:     "   ",
			baseURL:  "https://example.com/",
			expected: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base, err := parseBaseURL(tt.baseURL)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, resolveIconURL(tt.href, base))
		})
	}
}

func TestLinkIconRegex(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "standard rel=icon",
			html:     `<link rel="icon" href="/favicon.png">`,
			expected: "/favicon.png",
		},
		{
			name:     "shortcut icon",
			html:     `<link rel="shortcut icon" href="/icon.ico">`,
			expected: "/icon.ico",
		},
		{
			name:     "apple-touch-icon",
			html:     `<link rel="apple-touch-icon" href="/apple-icon.png">`,
			expected: "/apple-icon.png",
		},
		{
			name:     "with type attribute",
			html:     `<link rel="icon" type="image/png" href="/icon.png">`,
			expected: "/icon.png",
		},
		{
			name:     "single quotes",
			html:     `<link rel='icon' href='/icon.svg'>`,
			expected: "/icon.svg",
		},
		{
			name:     "no match",
			html:     `<link rel="stylesheet" href="/style.css">`,
			expected: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			matches := linkIconRegex.FindSubmatch([]byte(tt.html))
			if tt.expected == "" {
				assert.Less(t, len(matches), 2)
			} else {
				require.GreaterOrEqual(t, len(matches), 2)
				assert.Equal(t, tt.expected, string(matches[1]))
			}
		})
	}
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
