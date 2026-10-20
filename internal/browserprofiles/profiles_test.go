package browserprofiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectForPlatformChromiumProfiles(t *testing.T) {
	home := t.TempDir()
	userData := filepath.Join(home, ".config", "google-chrome")
	require.NoError(t, os.MkdirAll(filepath.Join(userData, "Default"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(userData, "Profile 1"), 0700))
	localState := `{"profile":{"info_cache":{"Default":{"name":"Personal"},"Profile 1":{"name":"Work"},"Profile 2":{"name":"Missing"}}}}`
	require.NoError(t, os.WriteFile(filepath.Join(userData, "Local State"), []byte(localState), 0600))

	profiles := DetectForPlatform("Google Chrome", "linux", home, func(string) string { return "" })

	require.Equal(t, []Profile{
		{Name: "Personal", ID: "Default", Type: TypeChromium, IsDefault: true},
		{Name: "Work", ID: "Profile 1", Type: TypeChromium},
	}, profiles)
}

func TestDetectForPlatformFirefoxProfiles(t *testing.T) {
	home := t.TempDir()
	profileDir := filepath.Join(home, ".mozilla", "firefox")
	workDir := filepath.Join(profileDir, "work.profile")
	personalDir := filepath.Join(home, "custom-firefox-profile")
	require.NoError(t, os.MkdirAll(workDir, 0700))
	require.NoError(t, os.MkdirAll(personalDir, 0700))
	profilesINI := "[Profile0]\nName=Work\nIsRelative=1\nPath=work.profile\nDefault=1\n" +
		"[Profile1]\nName=Personal\nIsRelative=0\nPath=" + personalDir + "\n" +
		"[Profile2]\nName=Unavailable\nIsRelative=1\nPath=missing.profile\n"
	require.NoError(t, os.MkdirAll(profileDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "profiles.ini"), []byte(profilesINI), 0600))

	profiles := DetectForPlatform("Firefox", "linux", home, func(string) string { return "" })

	require.Equal(t, []Profile{
		{Name: "Work", ID: workDir, Type: TypeFirefox, IsDefault: true},
		{Name: "Personal", ID: personalDir, Type: TypeFirefox},
	}, profiles)
}

func TestChromiumProfileDirsUsePlatformLocations(t *testing.T) {
	home := t.TempDir()
	windowsDirs := chromiumProfileDirs("Google Chrome", "windows", home, func(string) string { return "C:/Users/test/AppData/Local" })
	require.Equal(t, []string{filepath.Join("C:/Users/test/AppData/Local", "Google", "Chrome", "User Data")}, windowsDirs)

	darwinDirs := chromiumProfileDirs("Google Chrome", "darwin", home, func(string) string { return "" })
	require.Equal(t, []string{filepath.Join(home, "Library", "Application Support", "Google", "Chrome")}, darwinDirs)

	linuxDirs := chromiumProfileDirs("Microsoft Edge", "linux", home, func(string) string { return "" })
	require.Equal(t, []string{filepath.Join(home, ".config", "microsoft-edge")}, linuxDirs)
}

func TestDetectForPlatformUsesPlatformLocations(t *testing.T) {
	home := t.TempDir()
	appData := filepath.Join(home, "local")
	userData := filepath.Join(appData, "Google", "Chrome", "User Data")
	require.NoError(t, os.MkdirAll(filepath.Join(userData, "Default"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(userData, "Local State"),
		[]byte(`{"profile":{"info_cache":{"Default":{"name":"Default"}}}}`), 0600))

	profiles := DetectForPlatform("Google Chrome", "windows", home, func(key string) string {
		if key == "LOCALAPPDATA" {
			return appData
		}
		return ""
	})

	require.Len(t, profiles, 1)
	assert.Equal(t, "Default", profiles[0].Name)
}

func TestPrivateArgument(t *testing.T) {
	for _, test := range []struct {
		browser string
		want    string
		ok      bool
	}{
		{browser: "Google Chrome", want: "--incognito", ok: true},
		{browser: "Microsoft Edge", want: "--incognito", ok: true},
		{browser: "Firefox", want: "-private-window", ok: true},
		{browser: "Safari", ok: false},
	} {
		t.Run(test.browser, func(t *testing.T) {
			got, ok := PrivateArgument(test.browser)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestDetectIgnoresUnsupportedBrowser(t *testing.T) {
	assert.Empty(t, DetectForPlatform("Safari", "darwin", t.TempDir(), func(string) string { return "" }))
}
