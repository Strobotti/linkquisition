package confighealth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/defangconfig"
)

func TestCheckFindsGoneCustomSourceFromLocalState(t *testing.T) {
	dir := t.TempDir()
	source := "https://example.test/hosts?token=secret"
	success := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	state, err := json.Marshal(map[string]interface{}{"gone": true, "lastSuccess": success})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "defang"), 0700))
	path := filepath.Join(dir, "defang", "source_"+defangconfig.SourceID(source)+".json")
	require.NoError(t, os.WriteFile(path, state, 0600))
	settings := &linkquisition.Settings{Plugins: []linkquisition.PluginSettings{{Path: "defang.so", Settings: map[string]interface{}{"sources": []string{"https://healthy.test/hosts", source}}}}}
	require.Equal(t, []Issue{{Code: IssueDefangSourceGone, PluginIndex: 0, SourceIndex: 1, LastSuccess: success}}, Check(settings, true, dir))
	settings.Plugins[0].Settings["sources"] = []string{"https://healthy.test/hosts"}
	require.Empty(t, Check(settings, true, dir))
}
