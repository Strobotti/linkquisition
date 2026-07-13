package main

import (
	"context"
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

func (p *unwrap) Metadata() linkquisition.PluginMetadata {
	return linkquisition.PluginMetadata{
		Name:        "Unwrap",
		Description: "Unwraps URLs that are wrapped inside redirect/tracking URLs (e.g. Microsoft Defender SafeLinks)",
		Author:      "Juha Jantunen",
		Version:     "2.0.0",
		URL:         "https://github.com/Strobotti/linkquisition",
		Settings: []linkquisition.PluginSettingDescriptor{
			{
				Key:             "rules",
				Label:           "Unwrap Rules",
				Description:     "List of match/parameter pairs defining which URLs to unwrap",
				Type:            linkquisition.SettingTypeKeyValueList,
				KeyField:        "match",
				KeyFieldLabel:   "Match (regex)",
				ValueField:      "parameter",
				ValueFieldLabel: "URL Parameter",
			},
			{
				Key:         "requireBrowserMatchToUnwrap",
				Label:       "Require Browser Match",
				Description: "Only unwrap if a browser rule matches the unwrapped URL",
				Type:        linkquisition.SettingTypeBool,
				Default:     false,
			},
		},
	}
}

func (p *unwrap) Setup(serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{}) error {
	var settings UnwrapPluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		return fmt.Errorf("error decoding settings: %w", err)
	}

	p.settings = settings
	p.serviceProvider = serviceProvider

	return nil
}

func (p *unwrap) ProcessURL(_ context.Context, u string) linkquisition.PluginResult {
	for _, rule := range p.settings.Rules {
		if rule.Match == "" {
			p.serviceProvider.GetLogger().Warn("empty match rule", "plugin", "unwrap")
			continue
		}

		if matches, _ := regexp.MatchString(rule.Match, u); matches {
			parsed, err := url.Parse(u)
			if err != nil {
				p.serviceProvider.GetLogger().Warn("error parsing query", "error", err.Error(), "plugin", "unwrap")
				return linkquisition.PluginResult{URL: u, Action: linkquisition.ActionContinue, ContinueChain: true}
			}

			if parsed.Query().Has(rule.Parameter) {
				newURL := parsed.Query().Get(rule.Parameter)
				p.serviceProvider.GetLogger().Debug(
					fmt.Sprintf("url modified `%s` => `%s`", u, newURL), "plugin", "unwrap",
				)

				if !p.settings.RequireBrowserMatchToUnwrap {
					p.serviceProvider.GetLogger().Debug("unwrapping URL without browser match", "plugin", "unwrap")
					return linkquisition.PluginResult{URL: newURL, Action: linkquisition.ActionContinue, ContinueChain: true}
				} else if browser, err := p.serviceProvider.GetSettings().GetMatchingBrowser(newURL); err == nil && browser != nil {
					p.serviceProvider.GetLogger().Debug(
						fmt.Sprintf(
							"found a matching browser-rule for browser `%s` with URL `%s`",
							browser.Name,
							newURL,
						), "plugin", "unwrap",
					)
					return linkquisition.PluginResult{URL: newURL, Action: linkquisition.ActionContinue, ContinueChain: true}
				}
			}
		}
	}

	return linkquisition.PluginResult{URL: u, Action: linkquisition.ActionContinue, ContinueChain: true}
}

func (p *unwrap) Shutdown(_ context.Context) {
	// no-op: unwrap has no background work
}

var Plugin unwrap
