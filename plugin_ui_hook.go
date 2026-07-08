package linkquisition

// PluginUIHook is an optional interface that plugins can implement to receive
// a callback when the browser picker window is shown. This allows plugins to
// add visual effects, overlays, or animations to the picker UI.
//
// The host application checks each loaded plugin for this interface using a type
// assertion. Plugins that do not implement it are simply skipped.
//
// The PickerCanvas abstraction is used instead of fyne.Window directly, so that
// plugins do not need to import fyne.io/fyne/v2 (which causes build-flag
// mismatches when the host binary is compiled with different ldflags or tags).
type PluginUIHook interface {
	// OnPickerShown is called after the browser picker window content is set
	// and before it is displayed. The plugin receives a PickerCanvas that
	// provides methods to add visual overlays.
	//
	// Implementations must not block — start goroutines for ongoing animations.
	// The window may be closed at any time; plugins should not panic if refresh
	// calls fail after the window is gone.
	OnPickerShown(canvas PickerCanvas)
}

// PickerCanvas provides a limited interface for plugins to draw overlays on
// the browser picker window. This abstraction avoids requiring plugins to
// import fyne.io/fyne/v2 directly.
type PickerCanvas interface {
	// AddRasterOverlay adds a pixel-based overlay to the picker window.
	// The draw function is called each time the overlay needs to be rendered.
	// It receives the width and height of the canvas and must return RGBA pixel
	// data (4 bytes per pixel: R, G, B, A) in row-major order.
	// Translucency is a value from 0.0 (opaque) to 1.0 (fully transparent).
	AddRasterOverlay(translucency float64, draw func(w, h int) []uint8)

	// ScheduleRefresh requests that the overlay be redrawn. Call this from a
	// goroutine after updating animation state. Safe to call from any goroutine.
	ScheduleRefresh()

	// Width returns the current canvas width in pixels.
	Width() int

	// Height returns the current canvas height in pixels.
	Height() int
}
