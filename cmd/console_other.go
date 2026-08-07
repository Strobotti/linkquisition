//go:build !windows

package main

// attachParentConsoleIfCLI is a no-op on non-Windows platforms.
// On Windows, this attaches to the parent console so CLI output is visible.
func attachParentConsoleIfCLI() {}
