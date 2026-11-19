package confighealth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/defangconfig"
)

func TestCheckDefangURLhaus(t *testing.T) {
	tests := []struct {
		name      string
		plugin    linkquisition.PluginSettings
		supported bool
		want      bool
	}{
		{"implicit defaults use working sources", linkquisition.PluginSettings{Path: "defang.so"}, true, false},
		{"empty list uses working defaults", linkquisition.PluginSettings{Path: "defang.so", Settings: map[string]interface{}{"sources": []interface{}{}}}, true, false},
		{"explicit default", linkquisition.PluginSettings{Path: "/plugins/defang.so", Settings: map[string]interface{}{"sources": []interface{}{defangconfig.URLhausHostFile, "https://example.test/hosts"}}}, true, true},
		{"go string slice", linkquisition.PluginSettings{Path: "defang", Settings: map[string]interface{}{"sources": []string{defangconfig.URLhausHostFile}}}, true, true},
		{"custom source", linkquisition.PluginSettings{Path: "defang.so", Settings: map[string]interface{}{"sources": []string{"https://example.test/hosts"}}}, true, false},
		{"disabled", linkquisition.PluginSettings{Path: "defang.so", IsDisabled: true}, true, false},
		{"unsupported", linkquisition.PluginSettings{Path: "defang.so"}, false, false},
		{"another plugin", linkquisition.PluginSettings{Path: "other.so"}, true, false},
		{"malformed list element", linkquisition.PluginSettings{Path: "defang.so", Settings: map[string]interface{}{"sources": []interface{}{[]interface{}{}}}}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &linkquisition.Settings{Plugins: []linkquisition.PluginSettings{tt.plugin}}
			issues := Check(settings, tt.supported, t.TempDir())
			if tt.want {
				require.Equal(t, []Issue{{Code: IssueDefangURLhaus, PluginIndex: 0}}, issues)
			} else {
				require.Empty(t, issues)
			}
		})
	}
}
