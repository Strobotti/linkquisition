package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/strobotti/linkquisition"
)

func TestResolvePluginPathFromDisk_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "sanitize.so")
	if err := os.WriteFile(pluginPath, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	got := resolvePluginPathFromDisk(pluginPath, "/nonexistent")
	if got != pluginPath {
		t.Errorf("expected %q, got %q", pluginPath, got)
	}
}

func TestResolvePluginPathFromDisk_AppendsSoExtension(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "sanitize.so")
	if err := os.WriteFile(pluginPath, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	// Pass without .so — should append it
	got := resolvePluginPathFromDisk(filepath.Join(dir, "sanitize"), "/nonexistent")
	if got != pluginPath {
		t.Errorf("expected %q, got %q", pluginPath, got)
	}
}

func TestResolvePluginPathFromDisk_FallsBackToPluginFolder(t *testing.T) {
	pluginDir := t.TempDir()
	pluginPath := filepath.Join(pluginDir, "defang.so")
	if err := os.WriteFile(pluginPath, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	got := resolvePluginPathFromDisk("defang", pluginDir)
	if got != pluginPath {
		t.Errorf("expected %q, got %q", pluginPath, got)
	}
}

func TestResolvePluginPathFromDisk_NotFound(t *testing.T) {
	got := resolvePluginPathFromDisk("nonexistent", "/tmp/no-such-dir")
	if got != "" {
		t.Errorf("expected empty string for not found, got %q", got)
	}
}

func TestBrowserMatchesFilter_EmptyFilter(t *testing.T) {
	b := linkquisition.BrowserSettings{Name: "Firefox"}
	if !browserMatchesFilter(b, "") {
		t.Error("empty filter should always match")
	}
}

func TestBrowserMatchesFilter_MatchesBrowserName(t *testing.T) {
	b := linkquisition.BrowserSettings{
		Name: "Google Chrome",
		Matches: []linkquisition.BrowserMatch{
			{Type: "site", Value: "unrelated.com"},
		},
	}

	if !browserMatchesFilter(b, "chrome") {
		t.Error("expected filter 'chrome' to match 'Google Chrome'")
	}
}

func TestBrowserMatchesFilter_MatchesRuleValue(t *testing.T) {
	b := linkquisition.BrowserSettings{
		Name: "Firefox",
		Matches: []linkquisition.BrowserMatch{
			{Type: "site", Value: "www.github.com"},
			{Type: "domain", Value: "reddit.com"},
		},
	}

	if !browserMatchesFilter(b, "github") {
		t.Error("expected filter 'github' to match rule value 'www.github.com'")
	}

	if !browserMatchesFilter(b, "reddit") {
		t.Error("expected filter 'reddit' to match rule value 'reddit.com'")
	}
}

func TestBrowserMatchesFilter_NoMatch(t *testing.T) {
	b := linkquisition.BrowserSettings{
		Name: "Firefox",
		Matches: []linkquisition.BrowserMatch{
			{Type: "site", Value: "www.example.com"},
		},
	}

	if browserMatchesFilter(b, "chromium") {
		t.Error("expected filter 'chromium' NOT to match Firefox or its rules")
	}
}

func TestBrowserMatchesFilter_CaseInsensitive(t *testing.T) {
	b := linkquisition.BrowserSettings{
		Name: "Microsoft Edge",
		Matches: []linkquisition.BrowserMatch{
			{Type: "site", Value: "www.Office365.com"},
		},
	}

	// The filter is expected to be pre-lowered by the caller (rebuildRulesList).
	// The function lowercases browser name and rule values for comparison.
	if !browserMatchesFilter(b, "edge") {
		t.Error("expected case-insensitive match on browser name")
	}

	if !browserMatchesFilter(b, "office365") {
		t.Error("expected case-insensitive match on rule value")
	}
}

func TestBrowserMatchesFilter_NoRules(t *testing.T) {
	b := linkquisition.BrowserSettings{
		Name:    "Safari",
		Matches: nil,
	}

	if !browserMatchesFilter(b, "safari") {
		t.Error("expected name match with no rules")
	}

	if browserMatchesFilter(b, "chrome") {
		t.Error("expected no match when name and rules don't match")
	}
}

func TestOpenExternalURLWithService_NotDefault(t *testing.T) {
	mock := &trackingBrowserService{isDefault: false}

	err := openExternalURLWithService("https://example.com", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.openDefaultCalls != 1 {
		t.Errorf("expected 1 call to OpenUrlWithDefaultBrowser, got %d", mock.openDefaultCalls)
	}
	if mock.openWithBrowserCalls != 0 {
		t.Errorf("expected 0 calls to OpenUrlWithBrowser, got %d", mock.openWithBrowserCalls)
	}
}

func TestOpenExternalURLWithService_IsDefault_UseFirstBrowser(t *testing.T) {
	mock := &trackingBrowserService{
		isDefault: true,
		browsers: []linkquisition.Browser{
			{Name: "Safari", Command: "com.apple.Safari"},
			{Name: "Firefox", Command: "org.mozilla.firefox"},
		},
	}

	err := openExternalURLWithService("https://example.com", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.openWithBrowserCalls != 1 {
		t.Errorf("expected 1 call to OpenUrlWithBrowser, got %d", mock.openWithBrowserCalls)
	}
	if mock.lastBrowserUsed != "Safari" {
		t.Errorf("expected Safari (first browser), got %q", mock.lastBrowserUsed)
	}
}

func TestOpenExternalURLWithService_IsDefault_NoBrowsers_Fallback(t *testing.T) {
	mock := &trackingBrowserService{
		isDefault: true,
		browsers:  nil, // no browsers available
	}

	err := openExternalURLWithService("https://example.com", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to OpenUrlWithDefaultBrowser
	if mock.openDefaultCalls != 1 {
		t.Errorf("expected fallback to OpenUrlWithDefaultBrowser, got %d calls", mock.openDefaultCalls)
	}
}

// trackingBrowserService records which methods were called.
type trackingBrowserService struct {
	isDefault            bool
	browsers             []linkquisition.Browser
	openDefaultCalls     int
	openWithBrowserCalls int
	lastBrowserUsed      string
}

func (m *trackingBrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	return m.browsers, nil
}

func (m *trackingBrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	return linkquisition.Browser{Name: "default"}, nil
}

func (m *trackingBrowserService) OpenUrlWithDefaultBrowser(_ string) error {
	m.openDefaultCalls++
	return nil
}

func (m *trackingBrowserService) OpenUrlWithBrowser(_ string, browser *linkquisition.Browser) error {
	m.openWithBrowserCalls++
	m.lastBrowserUsed = browser.Name
	return nil
}

func (m *trackingBrowserService) AreWeTheDefaultBrowser() bool { return m.isDefault }

func (m *trackingBrowserService) MakeUsTheDefaultBrowser() error { return nil }

func (m *trackingBrowserService) GetIconForBrowser(_ linkquisition.Browser) ([]byte, error) {
	return nil, nil
}
