package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeInputURL_HTTPURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https URL passes through unchanged",
			input:    "https://www.example.com/path?q=1",
			expected: "https://www.example.com/path?q=1",
		},
		{
			name:     "http URL passes through unchanged",
			input:    "http://example.com",
			expected: "http://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeInputURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeInputURL_FileURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "file:// URL passes through unchanged",
			input:    "file:///tmp/test.html",
			expected: "file:///tmp/test.html",
		},
		{
			name:     "file:// URL with spaces passes through unchanged",
			input:    "file:///tmp/my%20file.html",
			expected: "file:///tmp/my%20file.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeInputURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeInputURL_AbsolutePaths(t *testing.T) {
	// Use a real temp directory to get a valid absolute path on any OS
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.html")
	spaceFile := filepath.Join(tmpDir, "my file.html")

	result := normalizeInputURL(testFile)
	assert.Equal(t, "file://"+testFile, result)

	result = normalizeInputURL(spaceFile)
	assert.Equal(t, "file://"+spaceFile, result)
}

func TestNormalizeInputURL_RelativePaths(t *testing.T) {
	// Create a temp file so os.Stat succeeds
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.html")
	err := os.WriteFile(testFile, []byte("<html></html>"), 0600)
	assert.NoError(t, err)

	// Change to the temp directory so relative path resolves
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	result := normalizeInputURL("test.html")

	// On macOS, filepath.Abs may resolve symlinks (e.g. /var -> /private/var)
	// so we just verify the result is a valid file:// URL pointing to the file
	assert.True(t, len(result) > len("file://"), "result should be a file:// URL")
	assert.True(t, result[:7] == "file://", "result should start with file://")
	assert.True(t, filepath.Base(result) == "test.html", "result should end with test.html")
}

func TestNormalizeInputURL_NonExistentRelativePath(t *testing.T) {
	// A relative path that doesn't exist on disk should be returned as-is
	result := normalizeInputURL("nonexistent-xyz-12345.html")
	assert.Equal(t, "nonexistent-xyz-12345.html", result)
}

func TestIsRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "URL with scheme is not a relative path",
			input:    "https://example.com",
			expected: false,
		},
		{
			name:     "file:// URL is not a relative path",
			input:    "file:///tmp/test.html",
			expected: false,
		},
		{
			name:     "non-existent file is not detected as relative path",
			input:    "nonexistent-xyz-99999.html",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRelativePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRelativePath_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exists.html")
	err := os.WriteFile(testFile, []byte("<html></html>"), 0600)
	assert.NoError(t, err)

	// Change to the temp dir so relative stat works
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	assert.True(t, isRelativePath("exists.html"))
}
