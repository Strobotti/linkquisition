// Package qrcode provides a thin wrapper around QR code generation for use in the UI.
package qrcode

import (
	"fmt"

	goqrcode "github.com/skip2/go-qrcode"
)

const (
	// DefaultSize is the default QR code image size in pixels.
	DefaultSize = 256
)

// Generate creates a QR code PNG image for the given content string.
// Returns the PNG bytes or an error if the content cannot be encoded.
func Generate(content string, size int) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("content must not be empty")
	}

	if size <= 0 {
		size = DefaultSize
	}

	png, err := goqrcode.Encode(content, goqrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	return png, nil
}
