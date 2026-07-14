//go:build windows

package main

// setPickerWindowAlwaysOnTop is a no-op on Windows.
// Fyne on Windows uses native Win32 windows; the GLFW_FLOATING hint is not
// directly exposed. A future improvement could use SetWindowPos with HWND_TOPMOST.
func setPickerWindowAlwaysOnTop() {}
