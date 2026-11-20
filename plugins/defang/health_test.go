package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
)

func TestDefangHealthCheck(t *testing.T) {
	dir := t.TempDir()
	plugin := newPlugin()
	checker, ok := plugin.(linkquisition.PluginHealthChecker)
	require.True(t, ok)

	issues, err := checker.HealthCheck(nil, dir)
	require.NoError(t, err)
	require.Empty(t, issues)

	source := "https://example.test/hosts?token=secret"
	cacheDir := filepath.Join(dir, "defang")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	state, err := json.Marshal(map[string]interface{}{"gone": true})
	require.NoError(t, err)
	cachePath := sourceCachePath(cacheDir, source)
	path := strings.TrimSuffix(cachePath, ".txt") + ".json"
	require.NoError(t, os.WriteFile(path, state, 0600))

	config := map[string]interface{}{"sources": []interface{}{"https://urlhaus.abuse.ch/downloads/hostfile/", source}}
	issues, err = checker.HealthCheck(config, dir)
	require.NoError(t, err)
	require.Len(t, issues, 2)
	require.Contains(t, issues[0].Message, "URLhaus")
	require.True(t, issues[0].ResetSettingsToDefaults)
	require.Contains(t, issues[1].Message, "example.test")
	require.False(t, issues[1].ResetSettingsToDefaults, "custom sources must not suggest resetting")
	require.NotContains(t, issues[1].Message, "secret")

	issues, err = checker.HealthCheck(map[string]interface{}{"sources": []string{"https://other.test"}}, dir)
	require.NoError(t, err)
	require.Empty(t, issues)

	require.NoError(t, os.WriteFile(path, []byte("not json"), 0600))
	// A corrupt state file is treated as no state rather than failing the whole check.
	issues, err = checker.HealthCheck(map[string]interface{}{"sources": []string{source}}, dir)
	require.NoError(t, err)
	require.Empty(t, issues)

	_, err = checker.HealthCheck(map[string]interface{}{"sources": []interface{}{map[string]interface{}{}}}, dir)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret")
}
