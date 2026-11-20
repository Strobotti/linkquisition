package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"

	"github.com/mitchellh/mapstructure"

	"github.com/strobotti/linkquisition"
)

const urlhausHostFile = "https://urlhaus.abuse.ch/downloads/hostfile/"

var _ linkquisition.PluginHealthChecker = (*defang)(nil)

// HealthCheck inspects the supplied configuration and local state without starting downloads.
func (p *defang) HealthCheck(config map[string]interface{}, configFolderPath string) ([]linkquisition.PluginHealthIssue, error) {
	var settings DefangPluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		return nil, fmt.Errorf("decode defang settings: %w", err)
	}
	sources := settings.Sources
	if len(sources) == 0 {
		sources = defaultSources
	}

	var issues []linkquisition.PluginHealthIssue
	for _, source := range sources {
		if source == urlhausHostFile {
			issues = append(issues, linkquisition.PluginHealthIssue{
				Message: "Defang's URLhaus host-file source is no longer available. Other sources may still be working. " +
					"Edit the sources to use a supported hosts-format list.",
				ResetSettingsToDefaults: true,
			})
			continue
		}
		path := filepath.Join(configFolderPath, cacheDirName, sourceFilePrefix+sourceID(source)+sourceStateExt)
		state, err := readSourceState(path)
		if err != nil {
			return nil, fmt.Errorf("read blocklist source status: %w", err)
		}
		if state.Gone {
			issues = append(issues, linkquisition.PluginHealthIssue{
				Message: fmt.Sprintf(
					"Defang source %s returned HTTP 410 and is no longer available. Other sources may still be working. "+
						"Edit the sources to remove or replace it.", describeSource(source),
				),
				// Only a built-in source can be fixed by resetting; resetting would discard a user's custom list.
				ResetSettingsToDefaults: slices.Contains(defaultSources, source),
			})
		}
	}
	return issues, nil
}

// describeSource identifies a source by host only, as the full URL may contain credentials or tokens.
func describeSource(source string) string {
	if parsed, err := url.Parse(source); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return "(unparseable URL)"
}
