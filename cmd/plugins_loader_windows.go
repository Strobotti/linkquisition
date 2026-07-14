//go:build windows

package main

// initPluginSupport is a no-op on Windows — the Go plugin package does not
// support Windows, so plugin-related commands and flags are not registered.
func initPluginSupport() {}
