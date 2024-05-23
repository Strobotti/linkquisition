package linkquisition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/strobotti/linkquisition"
)

func TestURL_GetDomain(t *testing.T) {
	for _, tt := range [...]struct {
		name      string
		url       string
		expected  string
		expectErr bool
	}{
		{
			name:     "domain is returned from full URL with path and query",
			url:      "https://www.example.com/path/is/here?and=here&is=something",
			expected: "example.com",
		},
		{
			name:     "subdomain is not www",
			url:      "https://sub.example.com/path/is/here",
			expected: "example.com",
		},
		{
			name:     "domain is returned even if there is no subdomain",
			url:      "https://example.com/path/is/here",
			expected: "example.com",
		},
		{
			name:     "domain is correctly returned even if TLD has multiple parts",
			url:      "https://www.example.co.uk/path/is/here",
			expected: "example.co.uk",
		},
		{
			name:     "domain is correctly returned even if the subdomain has multiple parts",
			url:      "https://sub.sub.example.com/path/is/here",
			expected: "example.com",
		},
		{
			name:     "domain is correctly returned even if the brits make your life a living hell",
			url:      "https://oh.im-special.example.co.uk/path/is/here",
			expected: "example.co.uk",
		},
		{
			name:     "ip address will be returned as is",
			url:      "https://127.0.0.1/path/is/here",
			expected: "127.0.0.1",
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				u := NewURL(tt.url)
				domain, err := u.GetDomain()
				if tt.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				assert.Equal(t, tt.expected, domain)
			},
		)
	}
}

func TestURL_GetSite(t *testing.T) {
	for _, tt := range [...]struct {
		name      string
		url       string
		expected  string
		expectErr bool
	}{
		{
			name:     "site is returned from full URL with path and query",
			url:      "https://www.example.com/path/is/here?and=here&is=something",
			expected: "www.example.com",
		},
		{
			name:     "subdomain is not www",
			url:      "https://sub.example.com/path/is/here",
			expected: "sub.example.com",
		},
		{
			name:     "site is returned even if there is no subdomain",
			url:      "https://example.com/path/is/here",
			expected: "example.com",
		},
		{
			name:     "site is correctly returned even if TLD has multiple parts",
			url:      "https://www.example.co.uk/path/is/here",
			expected: "www.example.co.uk",
		},
		{
			name:     "site is correctly returned even if the subdomain has multiple parts",
			url:      "https://sub.sub.example.com/path/is/here",
			expected: "sub.sub.example.com",
		},
		{
			name:     "site is correctly returned even if the brits make your life a living hell",
			url:      "https://oh.im-special.example.co.uk/path/is/here",
			expected: "oh.im-special.example.co.uk",
		},
		{
			name:     "ip address will be returned as is",
			url:      "https://1.2.3.4/path/is/here",
			expected: "1.2.3.4",
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				u := NewURL(tt.url)
				site, err := u.GetSite()
				if tt.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				assert.Equal(t, tt.expected, site)
			},
		)
	}
}
