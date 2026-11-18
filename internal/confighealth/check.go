// Package confighealth detects known configuration problems without network access.
package confighealth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/defangconfig"
)

const (
	IssueDefangURLhaus    = "defang_urlhaus_gone"
	IssueDefangSourceGone = "defang_source_gone"
)

type Issue struct {
	Code        string
	PluginIndex int
	SourceIndex int
	LastSuccess time.Time
}

// Check reports known actionable configuration issues without network access.
// pluginsSupported must be false on platforms where plugins are unavailable.
func Check(settings *linkquisition.Settings, pluginsSupported bool, configDir string) []Issue {
	if settings == nil || !pluginsSupported {
		return nil
	}
	var issues []Issue
	for i, plugin := range settings.Plugins {
		if plugin.IsDisabled || !strings.EqualFold(strings.TrimSuffix(filepath.Base(plugin.Path), ".so"), "defang") {
			continue
		}
		sources, ok := sourceList(plugin.Settings["sources"])
		if !ok {
			continue
		} // Invalid settings are handled by plugin setup.
		for j, source := range sources {
			if source == defangconfig.URLhausHostFile {
				issues = append(issues, Issue{Code: IssueDefangURLhaus, PluginIndex: i, SourceIndex: j})
				continue
			}
			path := filepath.Join(configDir, "defang", "source_"+defangconfig.SourceID(source)+".json")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var state struct {
				Gone        bool      `json:"gone"`
				LastSuccess time.Time `json:"lastSuccess"`
			}
			if json.Unmarshal(data, &state) == nil && state.Gone {
				issues = append(issues, Issue{Code: IssueDefangSourceGone, PluginIndex: i, SourceIndex: j, LastSuccess: state.LastSuccess})
			}
		}
	}
	return issues
}

func sourceList(value interface{}) ([]string, bool) {
	if value == nil {
		return defangconfig.DefaultSources(), true
	}
	var result []string
	switch sources := value.(type) {
	case []string:
		result = sources
	case []interface{}:
		for _, source := range sources {
			text, ok := source.(string)
			if !ok {
				return nil, false
			}
			result = append(result, text)
		}
	default:
		return nil, false
	}
	if len(result) == 0 {
		return defangconfig.DefaultSources(), true
	}
	return result, true
}
