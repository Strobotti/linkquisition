package main_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"

	"github.com/strobotti/linkquisition/mock"
	. "github.com/strobotti/linkquisition/plugins/terminus"
)

func TestTerminus_Setup_Defaults(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	err := testedPlugin.Setup(provider, map[string]interface{}{})
	require.NoError(t, err)

	assert.Equal(t, 5, testedPlugin.MaxRedirects)
	assert.Equal(t, 2000*time.Millisecond, testedPlugin.RequestTimeout)
}

func TestTerminus_Setup_CustomSettings(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"maxRedirects":   10,
		"requestTimeout": "5s",
	})
	require.NoError(t, err)

	assert.Equal(t, 10, testedPlugin.MaxRedirects)
	assert.Equal(t, 5*time.Second, testedPlugin.RequestTimeout)
}

func TestTerminus_Setup_MalformedTimeout(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"requestTimeout": "not-a-duration",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requestTimeout is malformed")
}

func TestTerminus_Setup_InvalidConfigType(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))
	provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")

	testedPlugin := Plugin
	// Pass a config with wrong types — mapstructure.Decode will produce an error
	err := testedPlugin.Setup(provider, map[string]interface{}{
		"maxRedirects": "not-an-int",
	})
	assert.Error(t, err)
}

func TestTerminus_ProcessURL(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name          string
		inputURL      string
		expectedURL   string
		locations     map[string]string
		responseCodes map[string]int
	}{
		{
			name:        "original url should be returned if no redirect is detected",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www.example.com/some/thing?here=again",
		},
		{
			name:        "a url from location -header should be returned if a redirect is detected",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www2.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "https://www2.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "a chain of redirects works as expected",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www3.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again":  "https://www2.example.com/some/thing?here=again",
				"https://www2.example.com/some/thing?here=again": "https://www3.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again":  http.StatusMultipleChoices,
				"https://www2.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "a chain of redirects is capped to 5 hops",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www6.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again":  "https://www2.example.com/some/thing?here=again",
				"https://www2.example.com/some/thing?here=again": "https://www3.example.com/some/thing?here=again",
				"https://www3.example.com/some/thing?here=again": "https://www4.example.com/some/thing?here=again",
				"https://www4.example.com/some/thing?here=again": "https://www5.example.com/some/thing?here=again",
				"https://www5.example.com/some/thing?here=again": "https://www6.example.com/some/thing?here=again",
				"https://www6.example.com/some/thing?here=again": "https://www7.example.com/some/thing?here=again",
				"https://www7.example.com/some/thing?here=again": "https://www8.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again":  http.StatusMultipleChoices,
				"https://www2.example.com/some/thing?here=again": http.StatusMultipleChoices,
				"https://www3.example.com/some/thing?here=again": http.StatusMultipleChoices,
				"https://www4.example.com/some/thing?here=again": http.StatusMultipleChoices,
				"https://www5.example.com/some/thing?here=again": http.StatusMultipleChoices,
				"https://www6.example.com/some/thing?here=again": http.StatusMultipleChoices,
				"https://www7.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "location is a relative path will not be resolved",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "/some/other/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "location is to the same exact host will not be resolved",
			inputURL:    "https://www.example.com/some/thing?here=again",
			expectedURL: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "https://www.example.com/some/other/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "location is to the same exact host but the scheme is upgraded from http to https will be resolved",
			inputURL:    "http://www.example.com/some/thing?here=again",
			expectedURL: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"http://www.example.com/some/thing?here=again": "https://www.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"http://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testedPlugin := Plugin

			provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{}, "")
			err := testedPlugin.Setup(provider, map[string]interface{}{})
			require.NoError(t, err)

			testedPlugin.Client.Transport = &mock.RoundTripper{
				RoundTripFunc: func(r *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{},
					}

					if code, ok := tt.responseCodes[r.URL.String()]; ok {
						resp.StatusCode = code
					}
					if location, ok := tt.locations[r.URL.String()]; ok {
						resp.Header.Set("Location", location)
					}

					return resp, nil
				},
			}

			result := testedPlugin.ProcessURL(context.Background(), tt.inputURL)
			assert.Equal(t, linkquisition.ActionContinue, result.Action)
			assert.True(t, result.ContinueChain)
			assert.Equal(t, tt.expectedURL, result.URL)
		})
	}
}

func TestTerminus_Metadata(t *testing.T) {
	testedPlugin := Plugin
	meta := testedPlugin.Metadata()

	assert.Equal(t, "Terminus", meta.Name)
	assert.NotEmpty(t, meta.Description)
	assert.Len(t, meta.Settings, 2)
}
