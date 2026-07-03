package main

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/strobotti/linkquisition"
)

// defaultParams is a list of well-known tracking/marketing query parameters that are stripped by default
var defaultParams = []string{
	// Google Analytics / Ads
	"utm_source",
	"utm_medium",
	"utm_campaign",
	"utm_term",
	"utm_content",
	"utm_id",
	"utm_source_platform",
	"utm_creative_format",
	"utm_marketing_tactic",
	"gclid",
	"gclsrc",
	"dclid",
	"gad_source",
	// Facebook / Meta
	"fbclid",
	"fb_action_ids",
	"fb_action_types",
	"fb_source",
	"fb_ref",
	// Microsoft
	"msclkid",
	// Twitter / X
	"twclickid",
	"twsrc",
	"tweetid",
	// HubSpot
	"_hsenc",
	"_hsmi",
	"__hssc",
	"__hstc",
	"__hsfp",
	"hsCtaTracking",
	// Mailchimp
	"mc_cid",
	"mc_eid",
	// Yandex
	"yclid",
	"ymclid",
	// Vero
	"vero_id",
	"vero_conv",
	// Marketo
	"mkt_tok",
	// Adobe
	"s_cid",
	// General tracking / affiliate
	"igshid",
	"si",
	"ref_src",
	"ref_url",
}

// SanitizePluginSettings holds the configuration for the sanitize plugin
type SanitizePluginSettings struct {
	// StripDefaults controls whether the built-in list of known tracking parameters is used (default: true)
	StripDefaults *bool `json:"stripDefaults,omitempty"`

	// ExtraParams is a list of additional exact parameter names to strip
	ExtraParams []string `json:"extraParams,omitempty"`

	// ExtraPatterns is a list of regex patterns to match parameter names against
	ExtraPatterns []string `json:"extraPatterns,omitempty"`

	// OnlyMatchingUrls is a regex pattern; if set, only URLs matching this pattern are sanitized
	OnlyMatchingUrls string `json:"onlyMatchingUrls,omitempty"`
}

var _ linkquisition.Plugin = (*sanitize)(nil)

// sanitize is a plugin that strips tracking/marketing query parameters from URLs
type sanitize struct {
	settings        SanitizePluginSettings
	stripDefaults   bool
	compiledExtra   []*regexp.Regexp
	urlFilter       *regexp.Regexp
	serviceProvider linkquisition.PluginServiceProvider
}

func (p *sanitize) Setup(serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{}) {
	p.serviceProvider = serviceProvider
	p.stripDefaults = true

	var settings SanitizePluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		serviceProvider.GetLogger().Warn("error decoding settings", "error", err.Error(), "plugin", "sanitize")
		return
	}

	p.settings = settings

	if settings.StripDefaults != nil {
		p.stripDefaults = *settings.StripDefaults
	}

	// Compile extra patterns
	for _, pattern := range settings.ExtraPatterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			serviceProvider.GetLogger().Warn(
				fmt.Sprintf("invalid regex pattern: %s", pattern),
				"error", err.Error(),
				"plugin", "sanitize",
			)
			continue
		}
		p.compiledExtra = append(p.compiledExtra, compiled)
	}

	// Compile URL filter
	if settings.OnlyMatchingUrls != "" {
		compiled, err := regexp.Compile(settings.OnlyMatchingUrls)
		if err != nil {
			serviceProvider.GetLogger().Warn(
				fmt.Sprintf("invalid onlyMatchingUrls regex: %s", settings.OnlyMatchingUrls),
				"error", err.Error(),
				"plugin", "sanitize",
			)
		} else {
			p.urlFilter = compiled
		}
	}
}

func (p *sanitize) ModifyUrl(address string) string {
	// If a URL filter is configured, only sanitize matching URLs
	if p.urlFilter != nil && !p.urlFilter.MatchString(address) {
		return address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		p.serviceProvider.GetLogger().Warn("error parsing URL", "error", err.Error(), "plugin", "sanitize")
		return address
	}

	query := parsed.Query()
	if len(query) == 0 {
		return address
	}

	modified := false
	for param := range query {
		if p.shouldStrip(param) {
			query.Del(param)
			modified = true
		}
	}

	if !modified {
		return address
	}

	parsed.RawQuery = query.Encode()

	newUrl := parsed.String()
	p.serviceProvider.GetLogger().Debug(fmt.Sprintf("url sanitized `%s` => `%s`", address, newUrl), "plugin", "sanitize")

	return newUrl
}

// shouldStrip returns true if the given parameter name should be removed
func (p *sanitize) shouldStrip(param string) bool {
	// Check the built-in default list
	if p.stripDefaults {
		for _, defaultParam := range defaultParams {
			if param == defaultParam {
				return true
			}
		}
	}

	// Check extra params (exact match)
	for _, extra := range p.settings.ExtraParams {
		if param == extra {
			return true
		}
	}

	// Check extra patterns (regex)
	for _, pattern := range p.compiledExtra {
		if pattern.MatchString(param) {
			return true
		}
	}

	return false
}

var Plugin sanitize
