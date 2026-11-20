package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	initialRetryDelay   = time.Hour
	maxRetryDelay       = 24 * time.Hour
	sourceLockFilePerms = 0600
	cacheDirName        = "defang"
	sourceFilePrefix    = "source_"
	sourceStateExt      = ".json"
)

// sourceState is stored separately from the last successful hosts file. It never
// contains the source URL, which may contain credentials or other secrets.
type sourceState struct {
	LastAttempt  time.Time `json:"lastAttempt"`
	LastSuccess  time.Time `json:"lastSuccess"`
	NextAttempt  time.Time `json:"nextAttempt"`
	Failures     int       `json:"failures"`
	HTTPStatus   int       `json:"httpStatus,omitempty"`
	Gone         bool      `json:"gone,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
}

func sourceID(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func (p *defang) sourcePaths(source string) (cachePath, statePath, lockPath string) {
	prefix := filepath.Join(p.cacheDir, sourceFilePrefix+sourceID(source))
	return prefix + ".txt", prefix + sourceStateExt, prefix + ".lock"
}

// readSourceState loads the persisted state of a source. A missing or corrupt state file
// yields an empty state: the state is only a cache of scheduling hints, and treating
// corruption as fatal would stop the source from ever being refreshed again.
func readSourceState(path string) (sourceState, error) {
	var state sourceState
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return sourceState{}, nil //nolint:nilerr // corrupt state is discarded and rewritten on the next attempt
	}
	return state, nil
}

// pruneSourceFiles removes cache, state and lock files that don't belong to any of the
// given sources (e.g. sources that were removed from the configuration), as well as the
// legacy index-based cache files from earlier versions.
func pruneSourceFiles(cacheDir string, sources []string) {
	keep := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		keep[sourceFilePrefix+sourceID(source)] = struct{}{}
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, sourceFilePrefix) || strings.HasSuffix(name, ".tmp") {
			continue
		}
		if _, ok := keep[strings.TrimSuffix(name, filepath.Ext(name))]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(cacheDir, name))
	}
}

func writeSourceState(path string, state *sourceState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, writeErr := file.Write(data); writeErr != nil {
		_ = file.Close()
		return writeErr
	}
	if closeErr := file.Close(); closeErr != nil {
		return closeErr
	}
	if renameErr := os.Rename(file.Name(), path); renameErr != nil {
		return renameErr
	}
	return nil
}

func retryDelay(failures int) time.Duration {
	delay := initialRetryDelay
	for i := 1; i < failures && delay < maxRetryDelay; i++ {
		delay *= 2
		if delay > maxRetryDelay {
			return maxRetryDelay
		}
	}
	return delay
}

func sourceDue(state *sourceState, cachePath string, interval time.Duration, now time.Time) bool {
	if state.Gone || now.Before(state.NextAttempt) {
		return false
	}
	if state.LastSuccess.IsZero() {
		return true
	}
	if _, err := os.Stat(cachePath); err != nil {
		return true
	}
	return !now.Before(state.LastSuccess.Add(interval))
}

type sourceDownloadResult struct {
	NotModified  bool
	ETag         string
	LastModified string
}

type sourceHTTPError struct{ status int }

func (e *sourceHTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.status) }
