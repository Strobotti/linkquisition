package safety

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	t.Run("google safe browsing", func(t *testing.T) {
		checker, err := NewChecker("google_safe_browsing", "test-key")
		require.NoError(t, err)
		assert.Equal(t, "Google Safe Browsing", checker.Name())
	})

	t.Run("virustotal", func(t *testing.T) {
		checker, err := NewChecker("virustotal", "test-key")
		require.NoError(t, err)
		assert.Equal(t, "VirusTotal", checker.Name())
	})

	t.Run("unknown provider", func(t *testing.T) {
		_, err := NewChecker("unknown", "test-key")
		assert.Error(t, err)
	})
}

func TestThreatLevelOrdering(t *testing.T) {
	assert.True(t, ThreatLevelSafe < ThreatLevelSuspicious)
	assert.True(t, ThreatLevelSuspicious < ThreatLevelDangerous)
}
