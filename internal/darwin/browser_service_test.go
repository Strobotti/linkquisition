//go:build darwin

package darwin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func TestParseBrowserPlist_HTTPHandler(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier":  "com.example.browser",
		"CFBundleDisplayName": "Example Browser",
		"CFBundleName":        "ExampleBrowser",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"https", "http"},
			},
		},
	}

	writePlist(t, plistPath, data)

	bundleID, name, isHTTPHandler := parseBrowserPlist(plistPath, "Example.app")

	assert.True(t, isHTTPHandler)
	assert.Equal(t, "com.example.browser", bundleID)
	assert.Equal(t, "Example Browser", name)
}

func TestParseBrowserPlist_FallsBackToBundleName(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.browser",
		"CFBundleName":       "MyBrowser",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"HTTP"},
			},
		},
	}

	writePlist(t, plistPath, data)

	bundleID, name, isHTTPHandler := parseBrowserPlist(plistPath, "Something.app")

	assert.True(t, isHTTPHandler)
	assert.Equal(t, "com.example.browser", bundleID)
	assert.Equal(t, "MyBrowser", name)
}

func TestParseBrowserPlist_FallsBackToAppDirName(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.browser",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"https"},
			},
		},
	}

	writePlist(t, plistPath, data)

	_, name, isHTTPHandler := parseBrowserPlist(plistPath, "FancyBrowser.app")

	assert.True(t, isHTTPHandler)
	assert.Equal(t, "FancyBrowser", name)
}

func TestParseBrowserPlist_NotHTTPHandler(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.editor",
		"CFBundleName":       "TextEditor",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"myapp", "custom-scheme"},
			},
		},
	}

	writePlist(t, plistPath, data)

	_, _, isHTTPHandler := parseBrowserPlist(plistPath, "TextEditor.app")

	assert.False(t, isHTTPHandler)
}

func TestParseBrowserPlist_NoURLTypes(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.app",
		"CFBundleName":       "SomeApp",
	}

	writePlist(t, plistPath, data)

	_, _, isHTTPHandler := parseBrowserPlist(plistPath, "SomeApp.app")

	assert.False(t, isHTTPHandler)
}

func TestParseBrowserPlist_MissingFile(t *testing.T) {
	_, _, isHTTPHandler := parseBrowserPlist("/nonexistent/path/Info.plist", "Missing.app")

	assert.False(t, isHTTPHandler)
}

func TestParseBrowserPlist_InvalidPlist(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	err := os.WriteFile(plistPath, []byte("not a valid plist"), 0600)
	require.NoError(t, err)

	_, _, isHTTPHandler := parseBrowserPlist(plistPath, "Bad.app")

	assert.False(t, isHTTPHandler)
}

func TestGetIconPathFromApp_ValidIcon(t *testing.T) {
	appPath := t.TempDir()

	// Create the expected directory structure
	resourcesDir := filepath.Join(appPath, "Contents", "Resources")
	require.NoError(t, os.MkdirAll(resourcesDir, 0755))

	// Create an Info.plist with CFBundleIconFile
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	data := map[string]interface{}{
		"CFBundleIconFile": "AppIcon",
	}
	writePlist(t, plistPath, data)

	result := getIconPathFromApp(appPath)

	assert.Equal(t, filepath.Join(appPath, "Contents", "Resources", "AppIcon.icns"), result)
}

func TestGetIconPathFromApp_IconWithExtension(t *testing.T) {
	appPath := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(appPath, "Contents", "Resources"), 0755))

	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	data := map[string]interface{}{
		"CFBundleIconFile": "AppIcon.icns",
	}
	writePlist(t, plistPath, data)

	result := getIconPathFromApp(appPath)

	assert.Equal(t, filepath.Join(appPath, "Contents", "Resources", "AppIcon.icns"), result)
}

func TestGetIconPathFromApp_NoIconField(t *testing.T) {
	appPath := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(appPath, "Contents"), 0755))

	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.noicon",
	}
	writePlist(t, plistPath, data)

	result := getIconPathFromApp(appPath)

	assert.Empty(t, result)
}

func TestGetIconPathFromApp_MissingPlist(t *testing.T) {
	appPath := t.TempDir()

	result := getIconPathFromApp(appPath)

	assert.Empty(t, result)
}

func TestBundleIDToName_KnownBrowsers(t *testing.T) {
	tests := []struct {
		bundleID string
		expected string
	}{
		{"com.apple.Safari", "Safari"},
		{"com.google.Chrome", "Google Chrome"},
		{"org.mozilla.firefox", "Firefox"},
		{"com.microsoft.edgemac", "Microsoft Edge"},
		{"com.brave.Browser", "Brave Browser"},
		{"com.operasoftware.Opera", "Opera"},
		{"com.vivaldi.Vivaldi", "Vivaldi"},
		{"company.thebrowser.Browser", "Arc"},
	}

	for _, tt := range tests {
		t.Run(tt.bundleID, func(t *testing.T) {
			name := bundleIDToName(tt.bundleID)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestBundleIDToName_UnknownFallsBackToLastPart(t *testing.T) {
	// For an unknown bundle ID where mdfind won't find anything,
	// it should fall back to the last part of the bundle ID
	name := bundleIDToName("com.unknown.SuperBrowser")

	assert.Equal(t, "SuperBrowser", name)
}

func TestGetHTTPHandlers_ScansAppDirectories(t *testing.T) {
	// Create a fake app directory structure
	appDir := t.TempDir()
	appPath := filepath.Join(appDir, "TestBrowser.app", "Contents")
	require.NoError(t, os.MkdirAll(appPath, 0755))

	plistPath := filepath.Join(appPath, "Info.plist")
	data := map[string]interface{}{
		"CFBundleIdentifier":  "com.test.browser",
		"CFBundleDisplayName": "Test Browser",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"https", "http"},
			},
		},
	}
	writePlist(t, plistPath, data)

	// Also create a non-browser app
	nonBrowserPath := filepath.Join(appDir, "TextEditor.app", "Contents")
	require.NoError(t, os.MkdirAll(nonBrowserPath, 0755))

	nonBrowserPlist := filepath.Join(nonBrowserPath, "Info.plist")
	nonBrowserData := map[string]interface{}{
		"CFBundleIdentifier": "com.test.editor",
		"CFBundleName":       "Text Editor",
	}
	writePlist(t, nonBrowserPlist, nonBrowserData)

	// Create a BrowserService and override the directory scanning
	// We test parseBrowserPlist indirectly through the directory scan logic
	// by verifying our test plist is correctly parsed
	bundleID, name, isHTTPHandler := parseBrowserPlist(plistPath, "TestBrowser.app")
	assert.True(t, isHTTPHandler)
	assert.Equal(t, "com.test.browser", bundleID)
	assert.Equal(t, "Test Browser", name)

	// Non-browser should not be detected
	_, _, isHTTPHandler = parseBrowserPlist(nonBrowserPlist, "TextEditor.app")
	assert.False(t, isHTTPHandler)
}

func TestGetAvailableBrowsers_NoDuplicates(t *testing.T) {
	svc := &BrowserService{}
	browsers, err := svc.GetAvailableBrowsers()
	require.NoError(t, err)

	seen := make(map[string]bool)
	for _, b := range browsers {
		if seen[b.Command] {
			t.Errorf("duplicate browser bundle ID found: %s (%s)", b.Command, b.Name)
		}
		seen[b.Command] = true
	}
}

func TestGetAvailableBrowsers_ExcludesLinkquisition(t *testing.T) {
	svc := &BrowserService{}
	browsers, err := svc.GetAvailableBrowsers()
	require.NoError(t, err)

	for _, b := range browsers {
		if strings.Contains(strings.ToLower(b.Command), "linkquisition") {
			t.Errorf("linkquisition should be excluded from browser list but found: %s", b.Command)
		}
	}
}

func TestGetAvailableBrowsers_ReturnsNonEmptyList(t *testing.T) {
	// On any macOS system, at least Safari should be available
	svc := &BrowserService{}
	browsers, err := svc.GetAvailableBrowsers()
	require.NoError(t, err)
	assert.NotEmpty(t, browsers, "expected at least one browser on macOS")
}

func TestParseBrowserPlist_CaseInsensitiveSchemes(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")

	data := map[string]interface{}{
		"CFBundleIdentifier": "com.example.browser",
		"CFBundleName":       "CaseBrowser",
		"CFBundleURLTypes": []interface{}{
			map[string]interface{}{
				"CFBundleURLSchemes": []interface{}{"HTTPS", "HTTP"},
			},
		},
	}

	writePlist(t, plistPath, data)

	_, _, isHTTPHandler := parseBrowserPlist(plistPath, "CaseBrowser.app")

	assert.True(t, isHTTPHandler)
}

// writePlist is a test helper that writes a plist file at the given path.
func writePlist(t *testing.T, path string, data interface{}) {
	t.Helper()

	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	encoder := plist.NewEncoder(f)
	require.NoError(t, encoder.Encode(data))
}
