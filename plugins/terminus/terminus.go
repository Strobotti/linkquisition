package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/strobotti/linkquisition"
)

type TerminusPluginSettings struct {
	MaxRedirects   int    `json:"maxRedirects"`
	RequestTimeout string `json:"requestTimeout"`
}

var _ linkquisition.Plugin = (*terminus)(nil)

type terminus struct {
	MaxRedirects    int
	Client          *http.Client
	RequestTimeout  time.Duration
	serviceProvider linkquisition.PluginServiceProvider
}

func (p *terminus) Metadata() linkquisition.PluginMetadata {
	return linkquisition.PluginMetadata{
		Name:        "Terminus",
		Description: "Resolves redirect chains to find the final destination URL before browser matching",
		Author:      "Juha Jantunen",
		Version:     "2.0.0",
		URL:         "https://github.com/Strobotti/linkquisition",
		Settings: []linkquisition.PluginSettingDescriptor{
			{
				Key:         "maxRedirects",
				Label:       "Max Redirects",
				Description: "Maximum number of redirect hops to follow",
				Type:        linkquisition.SettingTypeInt,
				Default:     5,
			},
			{
				Key:         "requestTimeout",
				Label:       "Request Timeout",
				Description: "Maximum time to spend following redirects (Go duration format, e.g. \"2s\")",
				Type:        linkquisition.SettingTypeDuration,
				Default:     "2s",
			},
		},
	}
}

func (p *terminus) Setup(serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{}) error {
	p.MaxRedirects = 5
	p.RequestTimeout = time.Millisecond * 2000

	var settings TerminusPluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		return fmt.Errorf("error decoding settings: %w", err)
	}

	if settings.MaxRedirects > 0 {
		p.MaxRedirects = settings.MaxRedirects
	}
	if settings.RequestTimeout != "" {
		timeout, err := time.ParseDuration(settings.RequestTimeout)
		if err != nil {
			return fmt.Errorf("requestTimeout is malformed: %w", err)
		}
		p.RequestTimeout = timeout
	}

	p.serviceProvider = serviceProvider
	p.Client = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return nil
}

func (p *terminus) ProcessURL(ctx context.Context, address string) linkquisition.PluginResult {
	modifiedURL := address

	ctx, cancel := context.WithTimeout(ctx, p.RequestTimeout)
	defer cancel()

	for i := 0; i < p.MaxRedirects; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodHead, modifiedURL, http.NoBody)
		req.Header.Set("User-Agent", "linkquisition")
		resp, err := p.Client.Do(req)
		if err != nil {
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("error requesting HEAD %s", modifiedURL),
				"error", err.Error(),
				"plugin", "terminus",
			)
			return linkquisition.PluginResult{URL: modifiedURL, Action: linkquisition.ActionContinue, ContinueChain: true}
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
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("no location-header for HEAD %s", modifiedURL), "plugin", "terminus",
			)
			break
		}

		// if the location is a relative path, we assume it's due to a missing authentication and just return the original URL
		if location[0] == '/' {
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("location is just a path for %s", modifiedURL),
				"location", location, "plugin", "terminus",
			)
			break
		}

		// if the location is to the same host, we assume it's due to a missing authentication
		prevURL, prevURLErr := url.Parse(modifiedURL)
		nextURL, nextURLErr := url.Parse(location)
		if prevURLErr != nil || nextURLErr != nil {
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("error parsing URLs for %s", modifiedURL),
				"location", location,
				"plugin", "terminus",
				"errors", errors.Join(prevURLErr, nextURLErr),
			)
			break
		}
		if prevURL.Host == nextURL.Host && prevURL.Scheme == nextURL.Scheme {
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("location is to the same host and scheme for %s", modifiedURL),
				"location", location, "plugin", "terminus",
			)
			break
		}

		p.serviceProvider.GetLogger().Debug(
			fmt.Sprintf("following a redirect for %s", modifiedURL),
			"location", location, "plugin", "terminus",
		)

		modifiedURL = location
	}

	return linkquisition.PluginResult{URL: modifiedURL, Action: linkquisition.ActionContinue, ContinueChain: true}
}

func (p *terminus) Shutdown(_ context.Context) {
	// no-op: terminus has no background work
}

var Plugin terminus
