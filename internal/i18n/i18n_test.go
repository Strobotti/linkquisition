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
