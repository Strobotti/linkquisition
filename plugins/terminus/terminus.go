package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.Plugin = (*terminus)(nil)

type terminus struct {
	MaxRedirects    int
	Client          *http.Client
	RequestTimeout  time.Duration
	serviceProvider linkquisition.PluginServiceProvider
}

func (p *terminus) Setup(serviceProvider linkquisition.PluginServiceProvider, _ map[string]interface{}) {
	p.serviceProvider = serviceProvider
	p.MaxRedirects = 5                         // TODO make configurable
	p.RequestTimeout = time.Millisecond * 2000 // TODO make configurable
	p.Client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (p *terminus) ModifyUrl(url string) string {
	newUrl := url

	ctx, cancel := context.WithTimeout(context.Background(), p.RequestTimeout)
	defer cancel()

	for i := 0; i < p.MaxRedirects; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodHead, newUrl, http.NoBody)
		req.Header.Set("User-Agent", "linkquisition")
		resp, err := p.Client.Do(req)
		if err != nil {
			p.serviceProvider.GetLogger().Warn(fmt.Sprintf("error requesting HEAD %s", newUrl), "error", err.Error(), "plugin", "terminus")
			return newUrl
		}

		if resp.Body != nil {
			_ = resp.Body.Close()
		}

		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			// we got a non-redirect response, so we have reached our final destination
			break
		}

		location := resp.Header.Get("Location")

		if location == "" {
			// for whatever reason the location -header doesn't contain a URL; skip
			p.serviceProvider.GetLogger().Warn(fmt.Sprintf("no location-header for HEAD %s", newUrl), "plugin", "terminus")
			break
		}

		p.serviceProvider.GetLogger().Debug(
			fmt.Sprintf("following a redirect for %s", newUrl), "location", location, "plugin", "terminus",
		)

		newUrl = location
	}

	return newUrl
}

var Plugin terminus
