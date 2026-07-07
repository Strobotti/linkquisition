package linkquisition_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/strobotti/linkquisition"
)

type testPathProvider struct {
	dir string
}

func (p *testPathProvider) GetConfigFolderPath() string { return p.dir }
func (p *testPathProvider) GetLogFolderPath() string    { return p.dir }
func (p *testPathProvider) GetPluginFolderPath() string { return p.dir }

type mockBrowserService struct {
	browsers []Browser
	err      error
}

func (m *mockBrowserService) GetAvailableBrowsers() ([]Browser, error)  { return m.browsers, m.err }
func (m *mockBrowserService) GetDefaultBrowser() (Browser, error)       { return Browser{}, nil }
func (m *mockBrowserService) OpenUrlWithDefaultBrowser(string) error    { return nil }
func (m *mockBrowserService) OpenUrlWithBrowser(string, *Browser) error { return nil }
func (m *mockBrowserService) AreWeTheDefaultBrowser() bool              { return false }
func (m *mockBrowserService) MakeUsTheDefaultBrowser() error            { return nil }
func (m *mockBrowserService) GetIconForBrowser(Browser) ([]byte, error) {
	return nil, nil
}

func newTestService(t *testing.T, browsers []Browser) *FileSettingsService {
	t.Helper()
	return &FileSettingsService{
		BrowserService: &mockBrowserService{browsers: browsers},
		PathProvider:   &testPathProvider{dir: t.TempDir()},
	}
}

func TestFileSettingsService_IsConfigured_ReturnsFalseWhenNoFile(t *testing.T) {
	svc := newTestService(t, nil)

	configured, err := svc.IsConfigured()

	assert.NoError(t, err)
	assert.False(t, configured)
}

func TestFileSettingsService_WriteAndReadSettings(t *testing.T) {
	svc := newTestService(t, nil)

	settings := &Settings{
		LogLevel: "debug",
		Browsers: []BrowserSettings{
			{Name: "Firefox", Command: "firefox", Source: SourceAuto},
		},
	}

	err := svc.WriteSettings(settings)
	require.NoError(t, err)

	read, err := svc.ReadSettings()
	require.NoError(t, err)
	assert.Equal(t, settings, read)
}

func TestFileSettingsService_IsConfigured_ReturnsTrueAfterWrite(t *testing.T) {
	svc := newTestService(t, nil)

	err := svc.WriteSettings(GetDefaultSettings())
	require.NoError(t, err)

	configured, err := svc.IsConfigured()

	assert.NoError(t, err)
	assert.True(t, configured)
}

func TestFileSettingsService_GetSettings_ReturnsDefaultWhenNotConfigured(t *testing.T) {
	svc := newTestService(t, nil)

	settings := svc.GetSettings()

	assert.Equal(t, GetDefaultSettings(), settings)
}

func TestFileSettingsService_GetSettings_ReturnsStoredSettings(t *testing.T) {
	svc := newTestService(t, nil)

	written := &Settings{
		LogLevel: "warn",
		Browsers: []BrowserSettings{
			{Name: "Chrome", Command: "chrome", Source: SourceManual},
		},
	}
	require.NoError(t, svc.WriteSettings(written))

	settings := svc.GetSettings()

	assert.Equal(t, written, settings)
}

func TestFileSettingsService_ScanBrowsers_CreatesConfig(t *testing.T) {
	browsers := []Browser{
		{Name: "Firefox", Command: "firefox"},
		{Name: "Chrome", Command: "chrome"},
	}
	svc := newTestService(t, browsers)

	err := svc.ScanBrowsers()
	require.NoError(t, err)

	configured, _ := svc.IsConfigured()
	assert.True(t, configured)

	settings, err := svc.ReadSettings()
	require.NoError(t, err)
	assert.Len(t, settings.Browsers, 2)
	assert.Equal(t, "Firefox", settings.Browsers[0].Name)
	assert.Equal(t, "Chrome", settings.Browsers[1].Name)
}

func TestFileSettingsService_ScanBrowsers_PreservesExistingRules(t *testing.T) {
	browsers := []Browser{
		{Name: "Firefox", Command: "firefox"},
	}
	svc := newTestService(t, browsers)

	// Write initial settings with a rule
	initial := &Settings{
		Browsers: []BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox",
				Source:  SourceAuto,
				Matches: []BrowserMatch{{Type: BrowserMatchTypeSite, Value: "example.com"}},
			},
		},
	}
	require.NoError(t, svc.WriteSettings(initial))

	// Re-scan should preserve the existing rule
	err := svc.ScanBrowsers()
	require.NoError(t, err)

	settings, _ := svc.ReadSettings()
	require.Len(t, settings.Browsers, 1)
	assert.Len(t, settings.Browsers[0].Matches, 1)
	assert.Equal(t, "example.com", settings.Browsers[0].Matches[0].Value)
}

func TestFileSettingsService_ReadSettings_ReturnsErrorForCorruptFile(t *testing.T) {
	svc := newTestService(t, nil)

	// Write garbage to the config file
	configPath := svc.GetConfigFilePath()
	require.NoError(t, os.WriteFile(configPath, []byte("{invalid json"), 0600))

	_, err := svc.ReadSettings()
	assert.Error(t, err)
}

func TestFileSettingsService_IsConfigured_ReturnsFalseForCorruptFile(t *testing.T) {
	svc := newTestService(t, nil)

	configPath := svc.GetConfigFilePath()
	require.NoError(t, os.WriteFile(configPath, []byte("not json at all"), 0600))

	configured, err := svc.IsConfigured()
	assert.False(t, configured)
	assert.Error(t, err)
}

func TestFileSettingsService_ReadSettings_CompilesRegexPatterns(t *testing.T) {
	svc := newTestService(t, nil)

	settings := &Settings{
		Browsers: []BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox",
				Source:  SourceAuto,
				Matches: []BrowserMatch{
					{Type: BrowserMatchTypeRegex, Value: `.*\.example\.com`},
				},
			},
		},
	}
	require.NoError(t, svc.WriteSettings(settings))

	read, err := svc.ReadSettings()
	require.NoError(t, err)

	// The regex should work after reading (compiled during ReadSettings)
	assert.True(t, read.Browsers[0].MatchesUrl("https://sub.example.com/page"))
}

func TestFileSettingsService_GetSettings_ReturnsDefaultForCorruptFile(t *testing.T) {
	svc := newTestService(t, nil)

	configPath := svc.GetConfigFilePath()
	require.NoError(t, os.WriteFile(configPath, []byte("corrupt!"), 0600))

	// Should not panic, should return defaults
	settings := svc.GetSettings()
	assert.Equal(t, GetDefaultSettings(), settings)
}

func TestFileSettingsService_ScanBrowsers_ReturnsErrorWhenGetAvailableBrowsersFails(t *testing.T) {
	svc := &FileSettingsService{
		BrowserService: &mockBrowserService{browsers: nil, err: fmt.Errorf("no browsers found")},
		PathProvider:   &testPathProvider{dir: t.TempDir()},
	}

	err := svc.ScanBrowsers()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan browsers")
}

func TestFileSettingsService_ScanBrowsers_ReturnsErrorWhenWriteFails(t *testing.T) {
	// Use a path that cannot be written to (non-existent nested directory with read-only parent)
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0555))

	svc := &FileSettingsService{
		BrowserService: &mockBrowserService{
			browsers: []Browser{{Name: "Firefox", Command: "firefox"}},
		},
		PathProvider: &testPathProvider{dir: filepath.Join(readOnlyDir, "subdir")},
	}

	err := svc.ScanBrowsers()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan browsers")
}

func TestFileSettingsService_ScanBrowsers_MergesWithExistingConfigOnRescan(t *testing.T) {
	browsers := []Browser{
		{Name: "Firefox", Command: "firefox"},
		{Name: "Chrome", Command: "chrome"},
	}
	svc := newTestService(t, browsers)

	// First scan creates config
	require.NoError(t, svc.ScanBrowsers())

	// Simulate a browser being uninstalled by changing the mock
	svc.BrowserService = &mockBrowserService{
		browsers: []Browser{{Name: "Firefox", Command: "firefox"}},
	}

	// Re-scan should drop Chrome (auto-added, no longer present)
	require.NoError(t, svc.ScanBrowsers())

	settings, err := svc.ReadSettings()
	require.NoError(t, err)
	assert.Len(t, settings.Browsers, 1)
	assert.Equal(t, "Firefox", settings.Browsers[0].Name)
}

func TestFileSettingsService_ScanBrowsers_PersistsIconPath(t *testing.T) {
	browsers := []Browser{
		{Name: "Firefox", Command: "firefox", IconPath: "/usr/share/icons/firefox.png"},
		{Name: "Chrome", Command: "chrome", IconPath: "/usr/share/icons/chrome.png"},
	}
	svc := newTestService(t, browsers)

	require.NoError(t, svc.ScanBrowsers())

	settings, err := svc.ReadSettings()
	require.NoError(t, err)
	assert.Len(t, settings.Browsers, 2)
	assert.Equal(t, "/usr/share/icons/firefox.png", settings.Browsers[0].IconPath)
	assert.Equal(t, "/usr/share/icons/chrome.png", settings.Browsers[1].IconPath)
}

func TestFileSettingsService_WriteSettings_ReturnsErrorForUnwritablePath(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0555))

	svc := &FileSettingsService{
		BrowserService: &mockBrowserService{},
		PathProvider:   &testPathProvider{dir: filepath.Join(readOnlyDir, "nested", "deep")},
	}

	err := svc.WriteSettings(GetDefaultSettings())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write settings")
}
