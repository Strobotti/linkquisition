package main

import (
	"testing"

	"github.com/strobotti/linkquisition"
)

func TestGetSettingsValue_Locale(t *testing.T) {
	settings := &linkquisition.Settings{Locale: "fi"}

	value, err := getSettingsValue(settings, "locale")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "fi" {
		t.Errorf("expected %q, got %q", "fi", value)
	}
}

func TestGetSettingsValue_LogLevel(t *testing.T) {
	settings := &linkquisition.Settings{LogLevel: "debug"}

	value, err := getSettingsValue(settings, "logLevel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "debug" {
		t.Errorf("expected %q, got %q", "debug", value)
	}
}

func TestGetSettingsValue_CaseInsensitive(t *testing.T) {
	settings := &linkquisition.Settings{LogLevel: "warn"}

	value, err := getSettingsValue(settings, "LOGLEVEL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "warn" {
		t.Errorf("expected %q, got %q", "warn", value)
	}
}

func TestGetSettingsValue_UnknownKey(t *testing.T) {
	settings := &linkquisition.Settings{}

	_, err := getSettingsValue(settings, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestSetSettingsValue_Locale(t *testing.T) {
	settings := &linkquisition.Settings{}

	if err := setSettingsValue(settings, "locale", "sv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if settings.Locale != "sv" {
		t.Errorf("expected locale %q, got %q", "sv", settings.Locale)
	}
}

func TestSetSettingsValue_LogLevel_Valid(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, level := range validLevels {
		settings := &linkquisition.Settings{}

		if err := setSettingsValue(settings, "logLevel", level); err != nil {
			t.Errorf("unexpected error for level %q: %v", level, err)
		}

		if settings.LogLevel != level {
			t.Errorf("expected logLevel %q, got %q", level, settings.LogLevel)
		}
	}
}

func TestSetSettingsValue_LogLevel_CaseInsensitiveValue(t *testing.T) {
	settings := &linkquisition.Settings{}

	if err := setSettingsValue(settings, "logLevel", "DEBUG"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if settings.LogLevel != "debug" {
		t.Errorf("expected logLevel %q, got %q", "debug", settings.LogLevel)
	}
}

func TestSetSettingsValue_LogLevel_Invalid(t *testing.T) {
	settings := &linkquisition.Settings{}

	err := setSettingsValue(settings, "logLevel", "verbose")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestSetSettingsValue_UnknownKey(t *testing.T) {
	settings := &linkquisition.Settings{}

	err := setSettingsValue(settings, "nonexistent", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestConfigGetSet_Integration(t *testing.T) {
	svc := newTestSettingsService(t)

	// Write initial settings
	settings := linkquisition.GetDefaultSettings()
	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write initial settings: %v", err)
	}

	// Read back and modify
	settings, err := svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	err = setSettingsValue(settings, "logLevel", "debug")
	if err != nil {
		t.Fatalf("failed to set logLevel: %v", err)
	}

	err = svc.WriteSettings(settings)
	if err != nil {
		t.Fatalf("failed to write modified settings: %v", err)
	}

	// Read again and verify
	settings, err = svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to re-read settings: %v", err)
	}

	value, err := getSettingsValue(settings, "logLevel")
	if err != nil {
		t.Fatalf("failed to get logLevel: %v", err)
	}

	if value != "debug" {
		t.Errorf("expected %q after round-trip, got %q", "debug", value)
	}
}
