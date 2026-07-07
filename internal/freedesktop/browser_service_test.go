//go:build linux

package freedesktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strobotti/linkquisition"
)

type mockBrowserIconLoader struct{}

func (m *mockBrowserIconLoader) LoadIcon(_ linkquisition.Browser) ([]byte, error) {
	return nil, nil
}

func (m *mockBrowserIconLoader) ResolveIconName(_ string) string {
	return ""
}

func TestDesktopEntryHasCategory_MatchesWebBrowser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "firefox.desktop")

	content := `[Desktop Entry]
Name=Firefox
Exec=firefox %u
Type=Application
Categories=Network;WebBrowser;
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.True(t, desktopEntryHasCategory(path, "WebBrowser"))
}

func TestDesktopEntryHasCategory_DoesNotMatchOtherCategory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "editor.desktop")

	content := `[Desktop Entry]
Name=Editor
Exec=editor %f
Type=Application
Categories=Development;TextEditor;
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.False(t, desktopEntryHasCategory(path, "WebBrowser"))
}

func TestDesktopEntryHasCategory_NoCategoriesLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.desktop")

	content := `[Desktop Entry]
Name=Minimal App
Exec=minimal
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.False(t, desktopEntryHasCategory(path, "WebBrowser"))
}

func TestDesktopEntryHasCategory_MissingFile(t *testing.T) {
	assert.False(t, desktopEntryHasCategory("/nonexistent/file.desktop", "WebBrowser"))
}

func TestDesktopEntryHasCategory_EmptyCategoriesLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.desktop")

	content := `[Desktop Entry]
Name=Empty
Exec=empty
Categories=
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.False(t, desktopEntryHasCategory(path, "WebBrowser"))
}

func TestDesktopEntryExecMatches_FullPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "firefox.desktop")

	content := `[Desktop Entry]
Name=Firefox
Exec=/usr/bin/firefox %u
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.True(t, desktopEntryExecMatches(path, "/usr/bin/firefox"))
}

func TestDesktopEntryExecMatches_BaseName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chrome.desktop")

	content := `[Desktop Entry]
Name=Chrome
Exec=google-chrome-stable %U
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.True(t, desktopEntryExecMatches(path, "/usr/bin/google-chrome-stable"))
}

func TestDesktopEntryExecMatches_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "firefox.desktop")

	content := `[Desktop Entry]
Name=Firefox
Exec=/usr/bin/firefox %u
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.False(t, desktopEntryExecMatches(path, "/usr/bin/chromium"))
}

func TestDesktopEntryExecMatches_NoExecLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.desktop")

	content := `[Desktop Entry]
Name=Broken
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	assert.False(t, desktopEntryExecMatches(path, "/usr/bin/broken"))
}

func TestDesktopEntryExecMatches_MissingFile(t *testing.T) {
	assert.False(t, desktopEntryExecMatches("/nonexistent/file.desktop", "/usr/bin/app"))
}

func TestGetApplicationPaths_UsesXDGDataDirs(t *testing.T) {
	// Create a temp directory structure
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	// Set XDG_DATA_DIRS to our test directory
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	paths := xdg.GetApplicationPaths()

	assert.Contains(t, paths, appsDir)
}

func TestGetApplicationPaths_DefaultsWhenUnset(t *testing.T) {
	// Unset XDG_DATA_DIRS — the function should use the defaults
	t.Setenv("XDG_DATA_DIRS", "")
	os.Unsetenv("XDG_DATA_DIRS")

	xdg := &XdgService{}
	paths := xdg.GetApplicationPaths()

	// On a typical system at least one of the default paths exists,
	// but in CI it might not. Just verify it doesn't panic.
	_ = paths
}

func TestGetDesktopEntryPathForFilename_Found(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	// Create a .desktop file
	desktopFile := filepath.Join(appsDir, "firefox.desktop")
	require.NoError(t, os.WriteFile(desktopFile, []byte("[Desktop Entry]\nName=Firefox\n"), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	result, err := xdg.GetDesktopEntryPathForFilename("firefox.desktop")

	assert.NoError(t, err)
	assert.Equal(t, desktopFile, result)
}

func TestGetDesktopEntryPathForFilename_NotFound(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	_, err := xdg.GetDesktopEntryPathForFilename("nonexistent.desktop")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .desktop entry found")
}

func TestGetDesktopEntryPathForBinary_Found(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	content := `[Desktop Entry]
Name=Firefox
Exec=/usr/bin/firefox %u
Type=Application
Categories=Network;WebBrowser;
`
	desktopFile := filepath.Join(appsDir, "firefox.desktop")
	require.NoError(t, os.WriteFile(desktopFile, []byte(content), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	result, err := xdg.GetDesktopEntryPathForBinary("/usr/bin/firefox")

	assert.NoError(t, err)
	assert.Equal(t, desktopFile, result)
}

func TestGetDesktopEntryPathForBinary_MatchesBasename(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	content := `[Desktop Entry]
Name=Chrome
Exec=google-chrome-stable %U
Type=Application
`
	desktopFile := filepath.Join(appsDir, "chrome.desktop")
	require.NoError(t, os.WriteFile(desktopFile, []byte(content), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	result, err := xdg.GetDesktopEntryPathForBinary("/usr/bin/google-chrome-stable")

	assert.NoError(t, err)
	assert.Equal(t, desktopFile, result)
}

func TestGetDesktopEntryPathForBinary_NotFound(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	content := `[Desktop Entry]
Name=Firefox
Exec=/usr/bin/firefox %u
Type=Application
`
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, "firefox.desktop"), []byte(content), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	_, err := xdg.GetDesktopEntryPathForBinary("/usr/bin/chromium")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .desktop entry found")
}

func TestGetAvailableBrowsers_FindsWebBrowsers(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	// Create a browser .desktop file
	browserContent := `[Desktop Entry]
Name=Firefox
Exec=/usr/bin/firefox %u
Type=Application
Categories=Network;WebBrowser;
Icon=firefox
`
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, "firefox.desktop"), []byte(browserContent), 0600))

	// Create a non-browser .desktop file
	editorContent := `[Desktop Entry]
Name=Editor
Exec=/usr/bin/editor
Type=Application
Categories=Development;TextEditor;
`
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, "editor.desktop"), []byte(editorContent), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	svc := &BrowserService{
		XdgService:          &XdgService{},
		DesktopEntryService: &DesktopEntryService{},
		BrowserIconLoader:   &mockBrowserIconLoader{},
	}

	browsers, err := svc.GetAvailableBrowsers()

	require.NoError(t, err)
	require.Len(t, browsers, 1)
	assert.Equal(t, "Firefox", browsers[0].Name)
	assert.Equal(t, "/usr/bin/firefox %u", browsers[0].Command)
}

func TestGetAvailableBrowsers_SkipsLinkquisition(t *testing.T) {
	dir := t.TempDir()
	appsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0755))

	content := `[Desktop Entry]
Name=Linkquisition
Exec=/usr/bin/linkquisition
Type=Application
Categories=Network;WebBrowser;
`
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, "linkquisition.desktop"), []byte(content), 0600))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	svc := &BrowserService{
		XdgService:          &XdgService{},
		DesktopEntryService: &DesktopEntryService{},
		BrowserIconLoader:   &mockBrowserIconLoader{},
	}

	browsers, err := svc.GetAvailableBrowsers()

	require.NoError(t, err)
	assert.Empty(t, browsers)
}

func TestGetAvailableBrowsers_EmptyPaths(t *testing.T) {
	// Point to a non-existent directory
	t.Setenv("XDG_DATA_HOME", "/nonexistent/local")
	t.Setenv("XDG_DATA_DIRS", "/nonexistent/path")

	svc := &BrowserService{
		XdgService:          &XdgService{},
		DesktopEntryService: &DesktopEntryService{},
		BrowserIconLoader:   &mockBrowserIconLoader{},
	}

	_, err := svc.GetAvailableBrowsers()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid desktop entry paths")
}

func TestExtractExecutable_FullPath(t *testing.T) {
	assert.Equal(t, "/usr/bin/firefox", extractExecutable("/usr/bin/firefox %u"))
}

func TestExtractExecutable_FullPathWithMultipleArgs(t *testing.T) {
	assert.Equal(t, "/usr/bin/firefox", extractExecutable("/usr/bin/firefox --name firefox-nightly -P nightly %u"))
}

func TestExtractExecutable_BasenameOnly(t *testing.T) {
	assert.Equal(t, "google-chrome-stable", extractExecutable("google-chrome-stable %U"))
}

func TestExtractExecutable_EmptyString(t *testing.T) {
	assert.Equal(t, "", extractExecutable(""))
}

func TestExtractExecutable_ExpandsTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "share", "firefox-dev", "firefox")
	assert.Equal(t, expected, extractExecutable("~/.local/share/firefox-dev/firefox --name firefox-dev %u"))
}

func TestExtractExecutable_ExpandsEnvVar(t *testing.T) {
	home, _ := os.UserHomeDir()
	t.Setenv("HOME", home)
	expected := filepath.Join(home, ".local", "share", "firefox-nightly", "firefox")
	assert.Equal(t, expected, extractExecutable("$HOME/.local/share/firefox-nightly/firefox --name firefox-nightly %u"))
}

func TestDesktopEntryExecMatches_CommandWithExtraArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "firefox-dev.desktop")

	// .desktop file has a simpler Exec= than the user's config
	content := `[Desktop Entry]
Name=Firefox Developer
Exec=/home/user/.local/share/firefox-dev/firefox --name firefox-dev %u
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	// User's config has extra args (-P nightly) not in the .desktop file
	// Should still match because the executable is the same
	assert.True(t, desktopEntryExecMatches(path, "/home/user/.local/share/firefox-dev/firefox --name firefox-dev -P nightly %u"))
}

func TestDesktopEntryExecMatches_EnvVarInCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "firefox-dev.desktop")

	home, _ := os.UserHomeDir()
	content := `[Desktop Entry]
Name=Firefox Developer
Exec=` + home + `/.local/share/firefox-dev/firefox --name firefox-dev %u
Type=Application
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	// Command uses $HOME which should expand to the same path
	t.Setenv("HOME", home)
	assert.True(t, desktopEntryExecMatches(path, "$HOME/.local/share/firefox-dev/firefox --name firefox-dev %u"))
}

func TestGetApplicationPaths_IncludesUserLocal(t *testing.T) {
	dir := t.TempDir()
	localAppsDir := filepath.Join(dir, "local", "applications")
	systemAppsDir := filepath.Join(dir, "share", "applications")
	require.NoError(t, os.MkdirAll(localAppsDir, 0755))
	require.NoError(t, os.MkdirAll(systemAppsDir, 0755))

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "local"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "share"))

	xdg := &XdgService{}
	paths := xdg.GetApplicationPaths()

	assert.Contains(t, paths, localAppsDir)
	assert.Contains(t, paths, systemAppsDir)
	// User-local should come first (higher priority)
	if len(paths) >= 2 {
		assert.Equal(t, localAppsDir, paths[0])
	}
}
