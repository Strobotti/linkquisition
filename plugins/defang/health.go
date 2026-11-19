package main

import (
	"fmt"
	"path/filepath"

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
	for i, source := range sources {
		if source == urlhausHostFile {
			issues = append(issues, linkquisition.PluginHealthIssue{
				Message: "Defang's URLhaus host-file source is no longer available. Other sources may still be working. " +
					"Edit the sources to use a supported hosts-format list.",
			})
			continue
		}
		path := filepath.Join(configFolderPath, cacheDirName, sourceFilePrefix+sourceID(source)+sourceStateExt)
		state, err := readSourceState(path)
		if err != nil {
			return nil, fmt.Errorf("read blocklist source status: %w", err)
		}
		if state.Gone {
			issues = append(issues, linkquisition.PluginHealthIssue{Message: fmt.Sprintf(
				"Defang source %d returned HTTP 410 and is no longer available. Other sources may still be working. "+
					"Edit the sources to remove or replace it.", i+1,
			)})
		}
	}
	return issues, nil
}
