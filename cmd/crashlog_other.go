//go:build !windows

package main

// redirectPanicLog is a no-op on non-Windows platforms.
// On Unix-like systems, stderr is always available for panic output.
func redirectPanicLog() {}
