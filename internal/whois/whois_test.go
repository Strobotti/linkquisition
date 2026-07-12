package whois

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "full URL with www",
			input:    "https://www.example.com/path?query=1",
			expected: "example.com",
		},
		{
			name:     "full URL without www",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "subdomain",
			input:    "https://sub.example.com/page",
			expected: "sub.example.com",
		},
		{
			name:     "URL without scheme",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "http scheme",
			input:    "http://example.org",
			expected: "example.org",
		},
		{
			name:     "URL with port",
			input:    "https://example.com:8080/path",
			expected: "example.com",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractDomain(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
