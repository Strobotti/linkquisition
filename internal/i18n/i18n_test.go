package i18n

import (
	"testing"
)

func TestInit_DefaultEnglish(t *testing.T) {
	Init("")

	got := T("picker.window_title")
	want := "Linkquisition"

	if got != want {
		t.Errorf("T(\"picker.window_title\") = %q, want %q", got, want)
	}
}

func TestInit_ExplicitOverride(t *testing.T) {
	Init("en")

	got := T("config.tab_general")
	want := "General"

	if got != want {
		t.Errorf("T(\"config.tab_general\") = %q, want %q", got, want)
	}
}

func TestT_WithTemplateData(t *testing.T) {
	Init("en")

	got := T("picker.remember_choice", map[string]interface{}{"Site": "example.com"})
	want := "Remember this choice with example.com"

	if got != want {
		t.Errorf("T(\"picker.remember_choice\") = %q, want %q", got, want)
	}
}

func TestT_UnknownMessageID(t *testing.T) {
	Init("en")

	got := T("nonexistent.message")
	want := "[nonexistent.message]"

	if got != want {
		t.Errorf("T(\"nonexistent.message\") = %q, want %q", got, want)
	}
}

func TestInit_UnsupportedLocale_FallsBackToEnglish(t *testing.T) {
	Init("xx")

	got := T("picker.window_title")
	want := "Linkquisition"

	if got != want {
		t.Errorf("T(\"picker.window_title\") with unsupported locale = %q, want %q", got, want)
	}
}

func TestInit_FinnishLocale(t *testing.T) {
	Init("fi")

	got := T("config.tab_general")
	want := "Yleiset"

	if got != want {
		t.Errorf("T(\"config.tab_general\") with fi locale = %q, want %q", got, want)
	}
}

func TestInit_FinnishLocale_WithTemplateData(t *testing.T) {
	Init("fi")

	got := T("picker.remember_choice", map[string]interface{}{"Site": "example.com"})
	want := "Muista tämä valinta sivustolle example.com"

	if got != want {
		t.Errorf("T(\"picker.remember_choice\") with fi locale = %q, want %q", got, want)
	}
}

func TestInit_SpanishLocale(t *testing.T) {
	Init("es")

	got := T("config.tab_about")
	want := "Acerca de"

	if got != want {
		t.Errorf("T(\"config.tab_about\") with es locale = %q, want %q", got, want)
	}
}

func TestInit_SwedishLocale(t *testing.T) {
	Init("sv")

	got := T("config.tab_general")
	want := "Allmänt"

	if got != want {
		t.Errorf("T(\"config.tab_general\") with sv locale = %q, want %q", got, want)
	}
}

func TestDetectLocale_OverrideTakesPrecedence(t *testing.T) {
	got := detectLocale("fi")
	want := "fi"

	if got != want {
		t.Errorf("detectLocale(\"fi\") = %q, want %q", got, want)
	}
}

func TestDetectLocale_EmptyOverrideFallsBack(t *testing.T) {
	got := detectLocale("")
	// Should return either the system locale or "en" — never empty
	if got == "" {
		t.Error("detectLocale(\"\") returned empty string")
	}
}

func TestAvailableLocales(t *testing.T) {
	locales := AvailableLocales()

	// Should contain at least the four known locales
	if len(locales) < 4 {
		t.Errorf("AvailableLocales() returned %d locales, want at least 4", len(locales))
	}

	expected := map[string]bool{"en": false, "es": false, "fi": false, "sv": false}
	for _, loc := range locales {
		if _, ok := expected[loc]; ok {
			expected[loc] = true
		}
	}

	for code, found := range expected {
		if !found {
			t.Errorf("AvailableLocales() missing expected locale %q", code)
		}
	}
}

func TestLocaleDisplayName(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{LocaleEnglish, "English"},
		{LocaleFinnish, "Suomi"},
		{LocaleSpanish, "Español"},
		{LocaleSwedish, "Svenska"},
		{"unknown", "unknown"}, // falls back to code itself
		{"", ""},               // empty code returns empty
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := LocaleDisplayName(tt.code)
			if got != tt.want {
				t.Errorf("LocaleDisplayName(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestTWithCount_FallbackOnMissingPluralForms(t *testing.T) {
	Init("en")

	// TWithCount returns the fallback format when the message doesn't define plural forms
	// This exercises the error handling path of TWithCount
	got := TWithCount("picker.window_title", 1)
	want := "[picker.window_title]"
	if got != want {
		t.Errorf("TWithCount(\"picker.window_title\", 1) = %q, want %q", got, want)
	}
}

func TestT_NilTemplateData(t *testing.T) {
	Init("en")

	// Passing nil as template data should not panic and should work normally
	got := T("picker.window_title", nil)
	want := "Linkquisition"
	if got != want {
		t.Errorf("T with nil template data = %q, want %q", got, want)
	}
}
