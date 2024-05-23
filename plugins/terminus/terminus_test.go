package main_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/strobotti/linkquisition"

	"github.com/strobotti/linkquisition/mock"
	. "github.com/strobotti/linkquisition/plugins/terminus"
)

func TestTerminus_ModifyUrl(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name          string
		inputUrl      string
		expectedUrl   string
		locations     map[string]string
		responseCodes map[string]int
	}{
		{
			name:        "original url should be returned if no redirect is detected",
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www.example.com/some/thing?here=again",
		},
		{
			name:        "a url from location -header should be returned if a redirect is detected",
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www2.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "https://www2.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "a chain of redirects works as expected",
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www3.example.com/some/thing?here=again",
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
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www6.example.com/some/thing?here=again",
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
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "/some/other/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "location is to the same exact host will not be resolved",
			inputUrl:    "https://www.example.com/some/thing?here=again",
			expectedUrl: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"https://www.example.com/some/thing?here=again": "https://www.example.com/some/other/thing?here=again",
			},
			responseCodes: map[string]int{
				"https://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
		{
			name:        "location is to the same exact host but the scheme is upgraded from http to https will be resolved",
			inputUrl:    "http://www.example.com/some/thing?here=again",
			expectedUrl: "https://www.example.com/some/thing?here=again",
			locations: map[string]string{
				"http://www.example.com/some/thing?here=again": "https://www.example.com/some/thing?here=again",
			},
			responseCodes: map[string]int{
				"http://www.example.com/some/thing?here=again": http.StatusMultipleChoices,
			},
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				testedPlugin := Plugin

				provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{})
				testedPlugin.Setup(provider, map[string]interface{}{})
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

				assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
			},
		)
	}
}
