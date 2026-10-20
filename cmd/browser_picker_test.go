package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func TestBrowserPickerPrivateModeEnabled(t *testing.T) {
	picker := &BrowserPicker{}
	if picker.privateModeEnabled() {
		t.Fatal("private mode should start disabled")
	}

	picker.handleShiftKey(&fyne.KeyEvent{Name: desktop.KeyShiftLeft}, true)
	if !picker.privateModeEnabled() {
		t.Fatal("holding left Shift should enable private mode")
	}

	picker.handleShiftKey(&fyne.KeyEvent{Name: desktop.KeyShiftRight}, true)
	picker.handleShiftKey(&fyne.KeyEvent{Name: desktop.KeyShiftLeft}, false)
	if !picker.privateModeEnabled() {
		t.Fatal("releasing one Shift key should not clear the other held Shift key")
	}

	picker.handleShiftKey(&fyne.KeyEvent{Name: desktop.KeyShiftRight}, false)
	if picker.privateModeEnabled() {
		t.Fatal("releasing both Shift keys should disable modifier private mode")
	}

	picker.privateToggleSet = true
	if !picker.privateModeEnabled() {
		t.Fatal("the private toggle should enable private mode independently of Shift")
	}
}
