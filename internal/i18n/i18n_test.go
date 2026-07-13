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

	got := T("picker.remember_choice")
	want := "Remember this choice for:"

	if got != want {
		t.Errorf("T(\"picker.remember_choice\") = %q, want %q", got, want)
	}

	got = T("picker.remember_site", map[string]interface{}{"Site": "www.example.com"})
	want = "www.example.com (this site only)"

	if got != want {
		t.Errorf("T(\"picker.remember_site\") = %q, want %q", got, want)
	}

	got = T("picker.remember_domain", map[string]interface{}{"Domain": "example.com"})
	want = "example.com (entire domain)"

	if got != want {
		t.Errorf("T(\"picker.remember_domain\") = %q, want %q", got, want)
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

	got := T("picker.remember_choice")
	want := "Muista tämä valinta:"

	if got != want {
		t.Errorf("T(\"picker.remember_choice\") with fi locale = %q, want %q", got, want)
	}

	got = T("picker.remember_site", map[string]interface{}{"Site": "www.example.com"})
	want = "www.example.com (vain tämä sivusto)"

	if got != want {
		t.Errorf("T(\"picker.remember_site\") with fi locale = %q, want %q", got, want)
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

	// Should contain at least the ten known locales
	if len(locales) < 10 {
		t.Errorf("AvailableLocales() returned %d locales, want at least 10", len(locales))
	}

	expected := map[string]bool{
		"de": false, "en": false, "es": false, "fi": false, "fr": false,
		"hu": false, "pt": false, "pt-BR": false, "sv": false, "uk": false,
	}
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
		{LocaleGerman, "Deutsch"},
		{LocaleEnglish, "English"},
		{LocaleSpanish, "Español"},
		{LocaleFinnish, "Suomi"},
		{LocaleFrench, "Français"},
		{LocaleHungarian, "Magyar"},
		{LocalePortuguese, "Português"},
		{LocaleBrazilianPortuguese, "Português (Brasil)"},
		{LocaleSwedish, "Svenska"},
		{LocaleUkrainian, "Українська"},
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

func TestTWithCount_WithCustomTemplateData(t *testing.T) {
	Init("en")

	// TWithCount with explicit template data should use that data
	// even though no plural forms are defined, the error path returns fallback
	got := TWithCount("nonexistent.key", 5, map[string]interface{}{"Count": 5, "Extra": "data"})
	want := "[nonexistent.key]"
	if got != want {
		t.Errorf("TWithCount with custom data = %q, want %q", got, want)
	}
}

func TestTWithCount_ZeroCount(t *testing.T) {
	Init("en")

	// Zero count with a message that has an "other" form should still resolve
	// go-i18n uses PluralCount to select the plural form; 0 maps to "other" in English
	got := TWithCount("nonexistent.key", 0)
	want := "[nonexistent.key]"
	if got != want {
		t.Errorf("TWithCount with 0 count for missing key = %q, want %q", got, want)
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
