//go:build windows

package windows

import "testing"

func TestExtractExePath(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "quoted executable with arguments",
			command: `"C:\Program Files\Browser\browser.exe" --open "%1"`,
			want:    `C:\Program Files\Browser\browser.exe`,
		},
		{
			name:    "unquoted executable with arguments",
			command: `C:\Browser\browser.exe --open "%1"`,
			want:    `C:\Browser\browser.exe`,
		},
		{
			name:    "quoted executable without arguments",
			command: `"C:\Program Files\Browser\browser.exe"`,
			want:    `C:\Program Files\Browser\browser.exe`,
		},
		{
			name:    "executable without arguments",
			command: `C:\Browser\browser.exe`,
			want:    `C:\Browser\browser.exe`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractExePath(tt.command); got != tt.want {
				t.Errorf("extractExePath(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
