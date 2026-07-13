//go:build linux

package main

// setPickerWindowAlwaysOnTop is a no-op on Linux.
// On Linux, window stacking is managed by the window manager and the .desktop file
// or wmctrl. Fyne's GLFW backend does not expose the GLFW_FLOATING hint.
func setPickerWindowAlwaysOnTop() {}
