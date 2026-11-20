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

func runWithSources(t *testing.T, dir string, sources ...string) linkquisition.Plugin {
	t.Helper()
	p := newPlugin()
	provider := linkquisition.NewPluginServiceProvider(newTestLogger(), &linkquisition.Settings{}, dir)
	require.NoError(t, p.Setup(provider, map[string]interface{}{"sources": sources}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Shutdown(ctx)
	return p
}

func TestSourceFailuresDoNotRefreshHealthySources(t *testing.T) {
	var gone, healthy atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gone":
			gone.Add(1)
			w.WriteHeader(http.StatusGone)
		case "/healthy":
			healthy.Add(1)
			_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		p := runWithSources(t, dir, server.URL+"/gone", server.URL+"/healthy")
		require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://malware.test").Action)
	}
	require.EqualValues(t, 1, gone.Load())
	require.EqualValues(t, 1, healthy.Load())
}

func TestTransientFailureIsBackedOffAcrossStarts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	dir := t.TempDir()
	runWithSources(t, dir, server.URL)
	runWithSources(t, dir, server.URL)
	require.EqualValues(t, 1, requests.Load())
	_, err := os.Stat(sourceCachePath(filepath.Join(dir, "defang"), server.URL))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestConcurrentStartsDoNotDuplicateDownloads(t *testing.T) {
	var requests atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		close(started)
		<-release
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	provider := linkquisition.NewPluginServiceProvider(newTestLogger(), &linkquisition.Settings{}, dir)
	first := newPlugin()
	require.NoError(t, first.Setup(provider, map[string]interface{}{"sources": []string{server.URL}}))
	<-started
	second := newPlugin()
	require.NoError(t, second.Setup(provider, map[string]interface{}{"sources": []string{server.URL}}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	second.Shutdown(ctx)
	close(release)
	first.Shutdown(ctx)
	require.EqualValues(t, 1, requests.Load())
	runWithSources(t, dir, server.URL)
	require.EqualValues(t, 1, requests.Load())
}

func TestSourceCacheFollowsSourceNotPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			_, _ = w.Write([]byte("0.0.0.0 first.test\n"))
		case "/second":
			_, _ = w.Write([]byte("0.0.0.0 second.test\n"))
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	runWithSources(t, dir, server.URL+"/first", server.URL+"/second")
	p := runWithSources(t, dir, server.URL+"/second")
	require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://second.test").Action)
	require.Equal(t, linkquisition.ActionContinue, p.ProcessURL(context.Background(), "https://first.test").Action)
}

func TestCorruptStateIsDiscardedAndRewritten(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	cachePath := sourceCachePath(cacheDir, server.URL)
	statePath := cachePath[:len(cachePath)-len(".txt")] + ".json"
	require.NoError(t, os.WriteFile(statePath, []byte("{not json"), 0600))

	p := runWithSources(t, dir, server.URL)

	require.EqualValues(t, 1, requests.Load())
	require.Equal(t, linkquisition.ActionBlock, p.ProcessURL(context.Background(), "https://malware.test").Action)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	require.True(t, json.Valid(data))
}

func TestInterruptedAttemptIsBackedOff(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	cachePath := sourceCachePath(cacheDir, server.URL)
	statePath := cachePath[:len(cachePath)-len(".txt")] + ".json"
	// A process that was killed mid-download leaves a recorded attempt with a pending backoff.
	state, err := json.Marshal(map[string]interface{}{
		"lastAttempt": time.Now(), "nextAttempt": time.Now().Add(time.Hour), "failures": 1,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, state, 0600))

	runWithSources(t, dir, server.URL)

	require.EqualValues(t, 0, requests.Load())
}

func TestUnusedSourceFilesAreRemoved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0.0.0.0 malware.test\n"))
	}))
	defer server.Close()
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	removedSource := sourceCachePath(cacheDir, "https://removed.test/hosts")
	legacy := filepath.Join(cacheDir, "source_0.txt")
	require.NoError(t, os.WriteFile(removedSource, []byte("0.0.0.0 old.test\n"), 0600))
	require.NoError(t, os.WriteFile(legacy, []byte("0.0.0.0 legacy.test\n"), 0600))

	p := runWithSources(t, dir, server.URL)

	require.NoFileExists(t, removedSource)
	require.NoFileExists(t, legacy)
	require.FileExists(t, sourceCachePath(cacheDir, server.URL))
	// Legacy index-based caches must never be applied to a different source.
	require.Equal(t, linkquisition.ActionContinue, p.ProcessURL(context.Background(), "https://legacy.test").Action)
}
