package main_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
)

func staleSourceState(t *testing.T, dir, source string) {
	t.Helper()
	path := sourceCachePath(filepath.Join(dir, "defang"), source)
	statePath := path[:len(path)-4] + ".json"
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var state map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &state))
	state["lastSuccess"] = time.Now().Add(-48 * time.Hour)
	data, err = json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, data, 0600))
}

func TestDefaultRefreshIntervalIs24Hours(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	runWithSources(t, dir, server.URL)
	staleSourceState(t, dir, server.URL)
	p := runWithSources(t, dir, server.URL)
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://malware.test").Action)
}

func TestConditionalRefreshNotModifiedKeepsCacheAndAdvancesFreshness(t *testing.T) {
	lastModified := "Wed, 18 Nov 2026 12:00:00 GMT"
	var requests atomic.Int32
	var unexpectedHeaders atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("If-None-Match") == `"version-1"` && r.Header.Get("If-Modified-Since") == lastModified {
			w.Header().Set("ETag", `"version-1"`)
			w.Header().Set("Last-Modified", lastModified)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Modified-Since") != "" {
			unexpectedHeaders.Store(true)
		}
		w.Header().Set("ETag", `"version-1"`)
		w.Header().Set("Last-Modified", lastModified)
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	runWithSources(t, dir, server.URL)
	staleSourceState(t, dir, server.URL)
	p := runWithSources(t, dir, server.URL)
	require.EqualValues(t, 2, requests.Load())
	require.False(t, unexpectedHeaders.Load())
	require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://malware.test").Action)

	statePath := sourceCachePath(filepath.Join(dir, "defang"), server.URL)
	statePath = statePath[:len(statePath)-4] + ".json"
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var state map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &state))
	require.Equal(t, `"version-1"`, state["etag"])
	require.Equal(t, lastModified, state["lastModified"])
	lastSuccess, err := time.Parse(time.RFC3339, state["lastSuccess"].(string))
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), lastSuccess, 5*time.Second)
}

func TestConditionalRefreshDownloadsChangedFeed(t *testing.T) {
	var requests atomic.Int32
	var missingValidator atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requests.Add(1)
		if count == 1 {
			w.Header().Set("ETag", `"version-1"`)
			_, _ = w.Write([]byte("0.0.0.0 old-malware.test\n"))
			return
		}
		if r.Header.Get("If-None-Match") != `"version-1"` {
			missingValidator.Store(true)
		}
		w.Header().Set("ETag", `"version-2"`)
		_, _ = w.Write([]byte("0.0.0.0 new-malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	runWithSources(t, dir, server.URL)
	staleSourceState(t, dir, server.URL)
	p := runWithSources(t, dir, server.URL)
	require.EqualValues(t, 2, requests.Load())
	require.False(t, missingValidator.Load())
	require.Equal(t, linkquisition.ActionContinue, p.ProcessURL(context.Background(), "https://old-malware.test").Action)
	require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://new-malware.test").Action)
}

func TestConditionalResponseWithoutCacheRetriesUnconditionally(t *testing.T) {
	var requests atomic.Int32
	var sentValidator atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("ETag", `"orphaned-validator"`)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.Header.Get("If-None-Match") != "" {
			sentValidator.Store(true)
		}
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(newTestLogger(), &linkquisition.Settings{}, dir)
	plugin := newPlugin()
	require.NoError(t, plugin.Setup(provider, map[string]interface{}{"sources": []string{server.URL}}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	plugin.Shutdown(ctx)
	require.EqualValues(t, 2, requests.Load())
	require.False(t, sentValidator.Load())
	require.Equal(t, linkquisition.ActionBlock, plugin.ProcessURL(context.Background(), "https://malware.test").Action)
}
