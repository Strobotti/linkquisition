package main

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/strobotti/linkquisition"
)

type UnwrapRule struct {
	// Match is a regular expression that the plugin should use to match URLs
	Match string `json:"match"`

	// Parameter is the query parameter that the plugin should use to unwrap URLs
	Parameter string `json:"parameter"`
}

// UnwrapPluginSettings is a struct that holds the settings specific for the unwrap plugin
type UnwrapPluginSettings struct {
	// Rules is a list of rules that the plugin should use to unwrap URLs
	Rules []UnwrapRule `json:"rules"`

	// RequireBrowserMatchToUnwrap is a boolean that determines if the plugin should only unwrap URLs if any browsers has a matching rule
	RequireBrowserMatchToUnwrap bool `json:"requireBrowserMatchToUnwrap,omitempty"`
}

var _ linkquisition.Plugin = (*unwrap)(nil)

// unwrap is a plugin that unwraps URLs based on the rules provided in the settings
type unwrap struct {
	settings        UnwrapPluginSettings
	serviceProvider linkquisition.PluginServiceProvider
}

func (p *unwrap) Setup(serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{}) {
	var settings UnwrapPluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		serviceProvider.GetLogger().Warn("error decoding settings", "error", err.Error(), "plugin", "unwrap")
	} else {
		p.settings = settings
	}

	p.serviceProvider = serviceProvider
}

func (p *unwrap) ModifyUrl(u string) string {
	for _, rule := range p.settings.Rules {
		if rule.Match == "" {
			p.serviceProvider.GetLogger().Warn("empty match rule", "plugin", "unwrap")
			continue
		}

		if matches, _ := regexp.MatchString(rule.Match, u); matches {
			parsed, err := url.Parse(u)
			if err != nil {
				p.serviceProvider.GetLogger().Warn("error parsing query", "error", err.Error(), "plugin", "unwrap")
				return u
			}

			if parsed.Query().Has(rule.Parameter) {
				newUrl := parsed.Query().Get(rule.Parameter)
				p.serviceProvider.GetLogger().Debug(fmt.Sprintf("url modified `%s` => `%s`", u, newUrl), "plugin", "unwrap")

				if !p.settings.RequireBrowserMatchToUnwrap {
					// the plugin is configured to always unwrap even if there's no matching browser-rule for final URL
					p.serviceProvider.GetLogger().Debug("unwrapping URL without browser match", "plugin", "unwrap")
					return newUrl
				} else if browser, err := p.serviceProvider.GetSettings().GetMatchingBrowser(newUrl); err == nil && browser != nil {
					// the plugin is configured to only unwrap if there's a matching browser-rule for final URL
					p.serviceProvider.GetLogger().Debug(
						fmt.Sprintf(
							"found a matching browser-rule for browser `%s` with URL `%s`",
							browser.Name,
							newUrl,
						), "plugin", "unwrap",
					)
					return newUrl
				}
			}
		}
	}

	return u
}

var Plugin unwrap
