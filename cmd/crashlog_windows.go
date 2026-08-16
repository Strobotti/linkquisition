//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
)

// redirectPanicLog opens a crash log file in the user's local app data folder
// and redirects the standard log package output there. This ensures panics and
// early startup errors are captured even when stdout/stderr are disconnected
// (GUI-subsystem binary launched from Explorer / Start Menu).
func redirectPanicLog() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	logDir := filepath.Join(cacheDir, "linkquisition", "logs")
	if mkErr := os.MkdirAll(logDir, 0755); mkErr != nil { //nolint:mnd
		return
	}

	crashLog := filepath.Join(logDir, "crash.log")

	f, openErr := os.OpenFile(crashLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600) //nolint:mnd
	if openErr != nil {
		return
	}

	log.SetOutput(f)
	log.Printf("linkquisition starting (version=%s, args=%v)", version, os.Args)
}
