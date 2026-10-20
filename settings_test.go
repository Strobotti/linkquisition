package linkquisition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/strobotti/linkquisition"
)

func TestMapSettingsLogLevelToSlog(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		input    string
		expected int
	}{
		{name: "debug", input: "debug", expected: -4},
		{name: "info", input: "info", expected: 0},
		{name: "warn", input: "warn", expected: 4},
		{name: "error", input: "error", expected: 8},
		{name: "unknown defaults to warn", input: "something", expected: 4},
		{name: "empty defaults to warn", input: "", expected: 4},
		{name: "case insensitive DEBUG", input: "DEBUG", expected: -4},
		{name: "case insensitive Warn", input: "Warn", expected: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := MapSettingsLogLevelToSlog(tt.input)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}
