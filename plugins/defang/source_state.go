package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/strobotti/linkquisition/internal/defangconfig"
)

const (
	initialRetryDelay = time.Hour
	maxRetryDelay     = 24 * time.Hour
)

// sourceState is stored separately from the last successful hosts file. It never
// contains the source URL, which may contain credentials or other secrets.
type sourceState struct {
	LastAttempt time.Time `json:"lastAttempt"`
	LastSuccess time.Time `json:"lastSuccess"`
	NextAttempt time.Time `json:"nextAttempt"`
	Failures    int       `json:"failures"`
	HTTPStatus  int       `json:"httpStatus,omitempty"`
	Gone        bool      `json:"gone,omitempty"`
}

func sourceID(source string) string {
	return defangconfig.SourceID(source)
}

func (p *defang) sourcePaths(source string) (string, string, string) {
	prefix := filepath.Join(p.cacheDir, "source_"+sourceID(source))
	return prefix + ".txt", prefix + ".json", prefix + ".lock"
}

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
		return state, err
	}
	return state, nil
}

func writeSourceState(path string, state sourceState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
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

func sourceDue(state sourceState, cachePath string, interval time.Duration, now time.Time) bool {
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

type sourceHTTPError struct{ status int }

func (e *sourceHTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.status) }
