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

func TestURL_GetDomain_ErrorCases(t *testing.T) {
	for _, tt := range [...]struct {
		name string
		url  string
	}{
		{
			name: "invalid URL returns error",
			url:  "://not-a-url",
		},
		{
			name: "empty string returns error",
			url:  "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u := NewURL(tt.url)
			_, err := u.GetDomain()
			assert.Error(t, err)
		})
	}
}

func TestURL_GetDomain_WithPorts(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "port is stripped from standard domain",
			url:      "https://www.example.com:443/path",
			expected: "example.com",
		},
		{
			name:     "port is stripped from localhost-like domain",
			url:      "http://myapp.local:8080/api",
			expected: "myapp.local",
		},
		{
			name:     "port is stripped from IP address",
			url:      "http://192.168.1.1:3000/admin",
			expected: "192.168.1.1",
		},
		{
			name:     "non-standard port on subdomain",
			url:      "https://api.example.com:8443/v1/resource",
			expected: "example.com",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u := NewURL(tt.url)
			domain, err := u.GetDomain()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, domain)
		})
	}
}

func TestURL_GetSite_WithPorts(t *testing.T) {
	for _, tt := range [...]struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "site includes port number",
			url:      "http://localhost:8080/path",
			expected: "localhost:8080",
		},
		{
			name:     "site includes explicit standard port",
			url:      "https://www.example.com:443/page",
			expected: "www.example.com:443",
		},
		{
			name:     "site includes non-standard port on IP",
			url:      "http://192.168.1.1:3000/api",
			expected: "192.168.1.1:3000",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u := NewURL(tt.url)
			site, err := u.GetSite()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, site)
		})
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
		{
			name:     "non-http URL returns empty string",
			url:      "ftp://files.example.com/data",
			expected: "",
		},
		{
			name:     "empty string returns empty",
			url:      "",
			expected: "",
		},
		{
			name:     "just a path returns empty",
			url:      "/some/path",
			expected: "",
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
