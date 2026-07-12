package qrcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("generates valid PNG for URL", func(t *testing.T) {
		png, err := Generate("https://example.com", DefaultSize)
		require.NoError(t, err)
		assert.NotEmpty(t, png)
		// PNG magic bytes
		assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, png[:4])
	})

	t.Run("uses default size when zero", func(t *testing.T) {
		png, err := Generate("https://example.com", 0)
		require.NoError(t, err)
		assert.NotEmpty(t, png)
	})

	t.Run("returns error for empty content", func(t *testing.T) {
		_, err := Generate("", DefaultSize)
		assert.Error(t, err)
	})

	t.Run("handles long URLs", func(t *testing.T) {
		longURL := "https://example.com/" + string(make([]byte, 500))
		png, err := Generate(longURL, DefaultSize)
		require.NoError(t, err)
		assert.NotEmpty(t, png)
	})
}
