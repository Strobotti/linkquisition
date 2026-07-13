package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{"same version", "2.13.0", "2.13.0", false},
		{"patch newer", "2.13.1", "2.13.0", true},
		{"minor newer", "2.14.0", "2.13.0", true},
		{"major newer", "3.0.0", "2.13.0", true},
		{"current is newer", "2.12.0", "2.13.0", false},
		{"dev build", "2.13.0", VersionDev, false},
		{"empty current", "2.13.0", "", false},
		{"pre-release suffix", "2.14.0-rc1", "2.13.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewerVersion(tt.latest, tt.current)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"2.13.0", []int{2, 13, 0}},
		{"0.1.0", []int{0, 1, 0}},
		{"10.20.30", []int{10, 20, 30}},
		{"1.0.0-rc1", []int{1, 0, 0}},
		{"invalid", nil},
		{"1.2", nil},
		{"a.b.c", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseSemver(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheck(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v2.14.0",
		HTMLURL: "https://github.com/Strobotti/linkquisition/releases/tag/v2.14.0",
		Name:    "v2.14.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Override the URL for testing
	origURL := githubReleasesURL
	t.Cleanup(func() { setGithubReleasesURL(origURL) })
	setGithubReleasesURL(server.URL)

	result, err := Check(context.Background(), "2.13.0")
	require.NoError(t, err)
	assert.Equal(t, "2.13.0", result.CurrentVersion)
	assert.Equal(t, "v2.14.0", result.LatestVersion)
	assert.True(t, result.IsNewer)
	assert.Contains(t, result.ReleaseURL, "v2.14.0")
}

func TestCheckUpToDate(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v2.13.0",
		HTMLURL: "https://github.com/Strobotti/linkquisition/releases/tag/v2.13.0",
		Name:    "v2.13.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	origURL := githubReleasesURL
	t.Cleanup(func() { setGithubReleasesURL(origURL) })
	setGithubReleasesURL(server.URL)

	result, err := Check(context.Background(), "2.13.0")
	require.NoError(t, err)
	assert.False(t, result.IsNewer)
}

func TestCheckHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origURL := githubReleasesURL
	t.Cleanup(func() { setGithubReleasesURL(origURL) })
	setGithubReleasesURL(server.URL)

	_, err := Check(context.Background(), "2.13.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}
