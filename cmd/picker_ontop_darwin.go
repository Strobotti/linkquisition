//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void SetKeyWindowFloating() {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *window = [[NSApplication sharedApplication] keyWindow];
		if (window != nil) {
			[window setLevel:NSFloatingWindowLevel];
		}
	});
}
*/
import "C"

// setPickerWindowAlwaysOnTop makes the currently focused window float above other windows.
// This uses the native Cocoa API to set NSFloatingWindowLevel on the key window.
func setPickerWindowAlwaysOnTop() {
	C.SetKeyWindowFloating()
}
