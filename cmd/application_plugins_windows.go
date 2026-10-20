//go:build windows

package main

import (
	"log/slog"

	"github.com/strobotti/linkquisition"
)

// loadApplicationPlugins returns no plugins because Go's plugin package is unsupported on Windows.
func loadApplicationPlugins(_ linkquisition.SettingsService, _ *slog.Logger) []linkquisition.Plugin {
	return nil
}
