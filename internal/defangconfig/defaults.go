// Package defangconfig contains the defaults shared by Defang and configuration checks.
package defangconfig

const URLhausHostFile = "https://urlhaus.abuse.ch/downloads/hostfile/"

// DefaultSources returns a fresh copy of the default blocklist URLs.
func DefaultSources() []string {
	return []string{
		URLhausHostFile,
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	}
}
