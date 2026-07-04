package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/strobotti/linkquisition"
)

// testPathProvider implements linkquisition.PathProvider for testing.
type testPathProvider struct {
	configFolder string
	logFolder    string
	pluginFolder string
}

func (p *testPathProvider) GetConfigFolderPath() string { return p.configFolder }
func (p *testPathProvider) GetLogFolderPath() string    { return p.logFolder }
func (p *testPathProvider) GetPluginFolderPath() string { return p.pluginFolder }

func newTestSettingsService(t *testing.T) *linkquisition.FileSettingsService {
	t.Helper()
	dir := t.TempDir()

	return &linkquisition.FileSettingsService{
		PathProvider: &testPathProvider{
			configFolder: dir,
			logFolder:    dir,
			pluginFolder: dir,
		},
	}
}

func TestRotateLogFile_NoFile(t *testing.T) {
	svc := newTestSettingsService(t)

	// Should not panic when the log file does not exist
	rotateLogFile(svc)
}

func TestRotateLogFile_SmallFile(t *testing.T) {
	svc := newTestSettingsService(t)
	logPath := svc.GetLogFilePath()

	// Create a small log file (under threshold)
	if err := os.WriteFile(logPath, []byte("small log content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rotateLogFile(svc)

	// Original file should still exist (not rotated)
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("expected log file to still exist, got: %v", err)
	}

	// No backup should exist
	if _, err := os.Stat(logPath + ".1"); !os.IsNotExist(err) {
		t.Error("expected no backup file for small log")
	}
}

func TestRotateLogFile_LargeFile(t *testing.T) {
	svc := newTestSettingsService(t)
	logPath := svc.GetLogFilePath()

	// Create a file larger than maxLogFileSize (1 MB)
	largeContent := make([]byte, maxLogFileSize+1)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	rotateLogFile(svc)

	// Original file should have been renamed to .1
	backupPath := logPath + ".1"
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("expected backup file to exist, got: %v", err)
	}
	if info.Size() != int64(maxLogFileSize+1) {
		t.Errorf("expected backup size %d, got %d", maxLogFileSize+1, info.Size())
	}

	// Original path should no longer exist
	if _, errStat := os.Stat(logPath); !os.IsNotExist(errStat) {
		t.Error("expected original log file to be gone after rotation")
	}
}

func TestRotateLogFile_OverwritesOldBackup(t *testing.T) {
	svc := newTestSettingsService(t)
	logPath := svc.GetLogFilePath()
	backupPath := logPath + ".1"

	// Create an existing backup
	if err := os.WriteFile(backupPath, []byte("old backup"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a large current log
	largeContent := make([]byte, maxLogFileSize+100)
	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	rotateLogFile(svc)

	// Backup should now be the new large file, not the old content
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(maxLogFileSize+100) {
		t.Errorf("expected backup to be overwritten with new content, got size %d", info.Size())
	}
}

func TestRotateLogFile_LogFilePath(t *testing.T) {
	svc := newTestSettingsService(t)
	logPath := svc.GetLogFilePath()

	// Verify the path ends with linkquisition.log
	expected := filepath.Join(svc.GetLogFolderPath(), "linkquisition.log")
	if logPath != expected {
		t.Errorf("expected log path %s, got %s", expected, logPath)
	}
}
