package freedesktop

import (
	"slices"
	"strings"
)

var launcherPrivateEnv = []string{
	"CHROME_DESKTOP",
	"CHROME_WRAPPER",
	"CHROME_VERSION_EXTRA",
	"GIO_LAUNCHED_DESKTOP_FILE",
	"DESKTOP_STARTUP_ID",
	"XDG_ACTIVATION_TOKEN",
	"ELECTRON_RUN_AS_NODE",
	"ELECTRON_FORCE_IS_PACKAGED",
}

// Returns a copy of the passed environment stripping some special values that
// were inherited from linkquistion's own caller.
//
// This is done to prevent an edge case where clicking a link in one electron
// app can cause the one the link should be opened in to retain instance
// specific env variables.
func sanitizeEnviron(environ []string) []string {
	return slices.DeleteFunc(slices.Clone(environ), func(entry string) bool {
		key, _, _ := strings.Cut(entry, "=")
		return slices.Contains(launcherPrivateEnv, key)
	})
}
