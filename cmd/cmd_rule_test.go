package main

import (
	"testing"

	"github.com/strobotti/linkquisition"
)

func newTestSettingsWithBrowsers() *linkquisition.Settings {
	return &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox %u",
				Source:  "auto",
				Matches: []linkquisition.BrowserMatch{
					{Type: "site", Value: "www.facebook.com"},
					{Type: "regex", Value: `.*\.reddit\.com`},
				},
			},
			{
				Name:    "Google Chrome",
				Command: "/usr/bin/google-chrome %U",
				Source:  "auto",
				Matches: []linkquisition.BrowserMatch{
					{Type: "domain", Value: "google.com"},
				},
			},
			{
				Name:    "Microsoft Edge",
				Command: "/usr/bin/microsoft-edge-stable %U",
				Source:  "auto",
			},
		},
	}
}

func TestFindBrowserByName_ExactMatch(t *testing.T) {
	settings := newTestSettingsWithBrowsers()

	idx, err := findBrowserByName(settings, "Firefox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindBrowserByName_CaseInsensitive(t *testing.T) {
	settings := newTestSettingsWithBrowsers()

	idx, err := findBrowserByName(settings, "firefox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindBrowserByName_PartialMatch(t *testing.T) {
	settings := newTestSettingsWithBrowsers()

	idx, err := findBrowserByName(settings, "edge")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 2 {
		t.Errorf("expected index 2 (Microsoft Edge), got %d", idx)
	}
}

func TestFindBrowserByName_NotFound(t *testing.T) {
	settings := newTestSettingsWithBrowsers()

	_, err := findBrowserByName(settings, "safari")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestFindBrowserByName_Ambiguous(t *testing.T) {
	settings := &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{Name: "Google Chrome"},
			{Name: "Google Chrome Beta"},
		},
	}

	_, err := findBrowserByName(settings, "chrome")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
}

func TestFindBrowserByName_ExactMatchOverridesPartial(t *testing.T) {
	settings := &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{Name: "Chrome"},
			{Name: "Google Chrome"},
		},
	}

	// "Chrome" should exact-match the first one, not be ambiguous
	idx, err := findBrowserByName(settings, "Chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx != 0 {
		t.Errorf("expected index 0 (exact match), got %d", idx)
	}
}

func TestRuleAdd_Integration(t *testing.T) {
	svc := newTestSettingsService(t)

	settings := newTestSettingsWithBrowsers()
	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Read, add rule, write
	settings, err := svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	idx, err := findBrowserByName(settings, "firefox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	settings.Browsers[idx].Matches = append(settings.Browsers[idx].Matches, linkquisition.BrowserMatch{
		Type:  "site",
		Value: "www.twitter.com",
	})

	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Read back and verify
	settings, err = svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to re-read settings: %v", err)
	}

	firefoxRules := settings.Browsers[0].Matches
	if len(firefoxRules) != 3 {
		t.Fatalf("expected 3 rules for Firefox, got %d", len(firefoxRules))
	}

	if firefoxRules[2].Type != "site" || firefoxRules[2].Value != "www.twitter.com" {
		t.Errorf("unexpected rule: %+v", firefoxRules[2])
	}
}

func TestRuleRemove_Integration(t *testing.T) {
	svc := newTestSettingsService(t)

	settings := newTestSettingsWithBrowsers()
	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Read, remove first rule from Firefox, write
	settings, err := svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	idx, err := findBrowserByName(settings, "firefox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Remove index 1 (0-based: 0)
	ruleIndex := 0
	settings.Browsers[idx].Matches = append(
		settings.Browsers[idx].Matches[:ruleIndex],
		settings.Browsers[idx].Matches[ruleIndex+1:]...,
	)

	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Read back and verify
	settings, err = svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to re-read settings: %v", err)
	}

	firefoxRules := settings.Browsers[0].Matches
	if len(firefoxRules) != 1 {
		t.Fatalf("expected 1 rule for Firefox after removal, got %d", len(firefoxRules))
	}

	if firefoxRules[0].Value != `.*\.reddit\.com` {
		t.Errorf("expected remaining rule to be the regex, got %+v", firefoxRules[0])
	}
}
