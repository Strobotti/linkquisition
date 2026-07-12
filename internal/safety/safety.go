// Package safety provides URL safety checking against threat intelligence services.
package safety

import (
	"context"
	"fmt"
	"time"
)

// ThreatLevel represents the severity of a URL safety check result.
type ThreatLevel int

const (
	// ThreatLevelSafe means no threats were detected.
	ThreatLevelSafe ThreatLevel = iota
	// ThreatLevelSuspicious means the URL has some risk indicators.
	ThreatLevelSuspicious
	// ThreatLevelDangerous means the URL is flagged as malicious.
	ThreatLevelDangerous
)

// CheckResult holds the outcome of a URL safety check.
type CheckResult struct {
	Level     ThreatLevel
	Provider  string
	Details   []string
	ReportURL string
	CheckedAt time.Time
}

// Checker defines the interface for URL safety checking providers.
type Checker interface {
	Check(ctx context.Context, url string) (*CheckResult, error)
	TestCredentials(ctx context.Context) error
	Name() string
}

// NewChecker creates a Checker for the given provider and API key.
func NewChecker(provider, apiKey string) (Checker, error) {
	switch provider {
	case "google_safe_browsing":
		return newGoogleSafeBrowsing(apiKey), nil
	case "virustotal":
		return newVirusTotal(apiKey), nil
	default:
		return nil, fmt.Errorf("unknown security provider: %s", provider)
	}
}
