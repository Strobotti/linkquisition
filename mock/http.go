package mock

import "net/http"

type RoundTripper struct {
	RoundTripFunc func(r *http.Request) (*http.Response, error)
}

func (rt *RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt.RoundTripFunc(r)
}
