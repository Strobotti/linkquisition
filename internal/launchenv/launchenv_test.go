package launchenv

import (
	"slices"
	"testing"
)

func TestSanitizeEnviron_RemovesTargetVars(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"PATH=/usr/bin:/bin",
		"CHROME_DESKTOP=signal-desktop.desktop",
		"ELECTRON_RUN_AS_NODE=1",
		"SHELL=/bin/bash",
	}

	got := SanitizeEnviron(input)

	expected := []string{
		"HOME=/home/user",
		"PATH=/usr/bin:/bin",
		"SHELL=/bin/bash",
	}

	if !slices.Equal(got, expected) {
		t.Errorf("SanitizeEnviron() = %v, want %v", got, expected)
	}
}

func TestSanitizeEnviron_CaseInsensitive(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"chrome_desktop=signal.desktop",
		"Electron_Run_As_Node=1",
	}

	got := SanitizeEnviron(input)

	expected := []string{
		"HOME=/home/user",
	}

	if !slices.Equal(got, expected) {
		t.Errorf("SanitizeEnviron() = %v, want %v", got, expected)
	}
}

func TestSanitizeEnviron_PreservesUnrelatedVars(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"PATH=/usr/bin",
		"DISPLAY=:0",
		"LANG=en_US.UTF-8",
		"XDG_RUNTIME_DIR=/run/user/1000",
	}

	got := SanitizeEnviron(input)

	if !slices.Equal(got, input) {
		t.Errorf("SanitizeEnviron() modified unrelated vars: got %v, want %v", got, input)
	}
}

func TestSanitizeEnviron_DoesNotMutateInput(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"CHROME_DESKTOP=signal.desktop",
		"PATH=/usr/bin",
	}

	original := slices.Clone(input)
	_ = SanitizeEnviron(input)

	if !slices.Equal(input, original) {
		t.Errorf("SanitizeEnviron() mutated the input slice: got %v, want %v", input, original)
	}
}

func TestSanitizeEnviron_EmptyInput(t *testing.T) {
	got := SanitizeEnviron([]string{})

	if len(got) != 0 {
		t.Errorf("SanitizeEnviron([]) = %v, want empty slice", got)
	}
}

func TestSanitizeEnviron_AllVarsStripped(t *testing.T) {
	input := []string{
		"ELECTRON_RUN_AS_NODE=1",
		"ELECTRON_FORCE_IS_PACKAGED=true",
		"CHROME_DESKTOP=signal-desktop.desktop",
		"CHROME_WRAPPER=/usr/bin/wrapper",
		"CHROME_VERSION_EXTRA=stable",
		"GIO_LAUNCHED_DESKTOP_FILE=/usr/share/applications/signal.desktop",
		"DESKTOP_STARTUP_ID=signal_TIME12345",
		"XDG_ACTIVATION_TOKEN=token123",
	}

	got := SanitizeEnviron(input)

	if len(got) != 0 {
		t.Errorf("SanitizeEnviron() should strip all target vars, got %v", got)
	}
}

func TestSanitizeEnviron_VarWithEqualsInValue(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"CHROME_DESKTOP=app=signal",
		"SOME_VAR=key=value=extra",
	}

	got := SanitizeEnviron(input)

	expected := []string{
		"HOME=/home/user",
		"SOME_VAR=key=value=extra",
	}

	if !slices.Equal(got, expected) {
		t.Errorf("SanitizeEnviron() = %v, want %v", got, expected)
	}
}
