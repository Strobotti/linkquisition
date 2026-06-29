//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#include "url_handler.h"
*/
import "C"

// URLChannel receives URLs sent to the app via macOS Apple Events.
var URLChannel = make(chan string, 1)

//export HandleURL
func HandleURL(u *C.char) {
	// Non-blocking send; drop if channel is full (shouldn't happen in practice)
	select {
	case URLChannel <- C.GoString(u):
	default:
	}
}

// RegisterURLHandler registers the Apple Event handler for kAEGetURL.
// Must be called before the UI event loop starts.
func RegisterURLHandler() {
	C.StartURLHandler()
}

// PumpEvents runs the macOS run loop briefly to allow pending Apple Events to be delivered.
func PumpEvents(seconds float64) {
	C.PumpEvents(C.double(seconds))
}
