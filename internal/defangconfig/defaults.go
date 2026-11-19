// Package defangconfig contains the defaults shared by Defang and configuration checks.
package defangconfig

import (
	"crypto/sha256"
	"encoding/hex"
)

// SourceID identifies a source without storing its potentially sensitive URL.
func SourceID(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

const URLhausHostFile = "https://urlhaus.abuse.ch/downloads/hostfile/"

// DefaultSources returns a fresh copy of the default blocklist URLs.
func DefaultSources() []string {
	return []string{
		"https://netapi.com/api2/?method=compromised&dataset_type=url&format=hosts",
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	}
}
