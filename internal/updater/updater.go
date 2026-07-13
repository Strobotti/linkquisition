// Package updater checks for newer releases on GitHub.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 10 * time.Second

// VersionDev is the version string used during development builds.
const VersionDev = "dev"

//nolint:gochecknoglobals // mutable for testing
var githubReleasesURL = "https://api.github.com/repos/Strobotti/linkquisition/releases/latest"

// setGithubReleasesURL overrides the API endpoint (used in tests).
func setGithubReleasesURL(url string) {
	githubReleasesURL = url
}

// ReleaseInfo holds information about the latest GitHub release.
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

// CheckResult holds the outcome of an update check.
type CheckResult struct {
	// CurrentVersion is the running version.
	CurrentVersion string

	// LatestVersion is the newest release tag on GitHub (e.g. "v2.13.0").
	LatestVersion string

	// ReleaseURL is the link to the release page.
	ReleaseURL string

	// IsNewer is true when LatestVersion is newer than CurrentVersion.
	IsNewer bool
}

// Check queries the GitHub releases API and compares the latest release tag
// against the running version. Returns a CheckResult or an error.
func Check(ctx context.Context, currentVersion string) (*CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleasesURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Linkquisition/"+currentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	cleanCurrent := strings.TrimPrefix(currentVersion, "v")

	return &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
		IsNewer:        isNewerVersion(latestVersion, cleanCurrent),
	}, nil
}

// isNewerVersion returns true if latest is a newer semver than current.
// Falls back to string comparison if parsing fails.
func isNewerVersion(latest, current string) bool {
	// Don't report updates for dev builds
	if current == VersionDev || current == "" {
		return false
	}

	latestParts := parseSemver(latest)
	currentParts := parseSemver(current)

	if latestParts == nil || currentParts == nil {
		return latest != current
	}

	for i := range 3 {
		if latestParts[i] > currentParts[i] {
			return true
		}

		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseSemver parses "major.minor.patch" into [3]int. Returns nil on failure.
func parseSemver(v string) []int {
	parts := strings.SplitN(v, ".", 3) //nolint:mnd
	if len(parts) != 3 {               //nolint:mnd
		return nil
	}

	result := make([]int, 3) //nolint:mnd

	for i, p := range parts {
		// Strip any pre-release suffix (e.g. "0-rc1")
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}

		var n int
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}

			n = n*10 + int(c-'0')
		}

		result[i] = n
	}

	return result
}
