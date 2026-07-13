package ui

import "image/color"

// Semantic UI colors used across the picker and configurator.
// These provide a consistent color palette for status indicators,
// validation feedback, and background highlights.
var (
	// ColorSuccess is used for positive indicators (safe, valid, confirmed).
	ColorSuccess = color.NRGBA{R: 50, G: 180, B: 50, A: 255}

	// ColorWarning is used for cautionary indicators (suspicious, check failed).
	ColorWarning = color.NRGBA{R: 220, G: 180, B: 50, A: 255}

	// ColorDanger is used for negative indicators (dangerous, invalid, error).
	ColorDanger = color.NRGBA{R: 220, G: 50, B: 50, A: 255}

	// ColorNeutral is used for inactive or indeterminate state indicators.
	ColorNeutral = color.NRGBA{R: 150, G: 150, B: 150, A: 255}

	// ColorHoverBg is a translucent highlight used for hover states.
	ColorHoverBg = color.NRGBA{R: 150, G: 150, B: 150, A: 30}

	// ColorAltRowBg is a very subtle background tint for alternating rows.
	ColorAltRowBg = color.NRGBA{R: 128, G: 128, B: 128, A: 15}
)
