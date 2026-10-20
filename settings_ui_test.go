package linkquisition_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/strobotti/linkquisition"
)

func TestUiSettings_GetPickerLayout(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		ui       UiSettings
		expected string
	}{
		{
			name:     "defaults to horizontal when empty",
			ui:       UiSettings{},
			expected: PickerLayoutHorizontal,
		},
		{
			name:     "returns vertical when explicitly set",
			ui:       UiSettings{PickerLayout: PickerLayoutVertical},
			expected: PickerLayoutVertical,
		},
		{
			name:     "returns horizontal when set",
			ui:       UiSettings{PickerLayout: PickerLayoutHorizontal},
			expected: PickerLayoutHorizontal,
		},
		{
			name:     "defaults to horizontal for unknown value",
			ui:       UiSettings{PickerLayout: "unknown"},
			expected: PickerLayoutHorizontal,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.ui.GetPickerLayout())
			},
		)
	}
}

func TestUiSettings_GetMaxItemsPerRow(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		ui       UiSettings
		expected int
	}{
		{
			name:     "defaults to DefaultMaxItemsPerRow when zero",
			ui:       UiSettings{},
			expected: DefaultMaxItemsPerRow,
		},
		{
			name:     "returns configured value when positive",
			ui:       UiSettings{MaxItemsPerRow: 3},
			expected: 3,
		},
		{
			name:     "defaults when negative",
			ui:       UiSettings{MaxItemsPerRow: -1},
			expected: DefaultMaxItemsPerRow,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.ui.GetMaxItemsPerRow())
			},
		)
	}
}

func TestUiSettings_GetTheme(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		ui       UiSettings
		expected string
	}{
		{
			name:     "defaults to system when empty",
			ui:       UiSettings{},
			expected: ThemeSystem,
		},
		{
			name:     "returns dark when set",
			ui:       UiSettings{Theme: ThemeDark},
			expected: ThemeDark,
		},
		{
			name:     "returns light when set",
			ui:       UiSettings{Theme: ThemeLight},
			expected: ThemeLight,
		},
		{
			name:     "returns system when explicitly set",
			ui:       UiSettings{Theme: ThemeSystem},
			expected: ThemeSystem,
		},
		{
			name:     "defaults to system for unknown value",
			ui:       UiSettings{Theme: "sepia"},
			expected: ThemeSystem,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.ui.GetTheme())
			},
		)
	}
}

func TestUiSettings_GetFaviconStrategy(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		ui       UiSettings
		expected string
	}{
		{
			name:     "defaults to direct when empty",
			ui:       UiSettings{},
			expected: FaviconStrategyDirect,
		},
		{
			name:     "returns direct when explicitly set",
			ui:       UiSettings{FaviconStrategy: FaviconStrategyDirect},
			expected: FaviconStrategyDirect,
		},
		{
			name:     "returns parsed when set",
			ui:       UiSettings{FaviconStrategy: FaviconStrategyParsed},
			expected: FaviconStrategyParsed,
		},
		{
			name:     "returns google when set",
			ui:       UiSettings{FaviconStrategy: FaviconStrategyGoogle},
			expected: FaviconStrategyGoogle,
		},
		{
			name:     "defaults to direct for unknown value",
			ui:       UiSettings{FaviconStrategy: "unknown"},
			expected: FaviconStrategyDirect,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.ui.GetFaviconStrategy())
			},
		)
	}
}

func TestSecurityCacheSettings_GetTTL(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		cache    SecurityCacheSettings
		expected time.Duration
	}{
		{
			name:     "defaults to 24h when TTLHours is zero",
			cache:    SecurityCacheSettings{},
			expected: 24 * time.Hour,
		},
		{
			name:     "returns configured TTL",
			cache:    SecurityCacheSettings{TTLHours: 12},
			expected: 12 * time.Hour,
		},
		{
			name:     "defaults to 24h for negative value",
			cache:    SecurityCacheSettings{TTLHours: -1},
			expected: 24 * time.Hour,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.cache.GetTTL())
			},
		)
	}
}
