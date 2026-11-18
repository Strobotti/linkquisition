// Package confighealth detects known configuration problems without network access.
package confighealth

import (
	"path/filepath"
	"strings"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/defangconfig"
)

const (
	IssueDefangURLhaus = "defang_urlhaus_gone"
)

type Issue struct {
	Code        string
	PluginIndex int
}

// Check reports known actionable configuration issues. pluginsSupported must be
// false on platforms where plugins are unavailable.
func Check(settings *linkquisition.Settings, pluginsSupported bool) []Issue {
	if settings == nil || !pluginsSupported {
		return nil
	}
	var issues []Issue
	for i, plugin := range settings.Plugins {
		if plugin.IsDisabled || !strings.EqualFold(strings.TrimSuffix(filepath.Base(plugin.Path), ".so"), "defang") {
			continue
		}
		sources, present := plugin.Settings["sources"]
		if !present || sources == nil {
			sources = defangconfig.DefaultSources()
		}
		switch list := sources.(type) {
		case []string:
			if len(list) == 0 {
				sources = defangconfig.DefaultSources()
			}
		case []interface{}:
			if len(list) == 0 {
				sources = defangconfig.DefaultSources()
			}
		}
		if hasURLhaus(sources) {
			issues = append(issues, Issue{Code: IssueDefangURLhaus, PluginIndex: i})
		}
	}
	return issues
}

func hasURLhaus(value interface{}) bool {
	switch sources := value.(type) {
	case []string:
		for _, source := range sources {
			if source == defangconfig.URLhausHostFile {
				return true
			}
		}
	case []interface{}:
		for _, source := range sources {
			if text, ok := source.(string); ok && text == defangconfig.URLhausHostFile {
				return true
			}
		}
	}
	return false
}
