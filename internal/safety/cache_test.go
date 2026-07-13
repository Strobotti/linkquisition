package safety

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_PutAndGet(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, "google_safe_browsing", 24*time.Hour)

	result := &CheckResult{
		Level:     ThreatLevelSafe,
		Provider:  "Google Safe Browsing",
		Details:   nil,
		ReportURL: "https://example.com/report",
		CheckedAt: time.Now(),
	}

	err := cache.Put("https://example.com", result)
	require.NoError(t, err)

	cached, ok := cache.Get("https://example.com")
	require.True(t, ok)
	assert.Equal(t, ThreatLevelSafe, cached.Level)
	assert.Equal(t, "Google Safe Browsing", cached.Provider)
	assert.Equal(t, "https://example.com/report", cached.ReportURL)
}

func TestCache_Miss(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, "virustotal", 24*time.Hour)

	cached, ok := cache.Get("https://not-cached.com")
	assert.False(t, ok)
	assert.Nil(t, cached)
}

func TestCache_Expired(t *testing.T) {
	cacheDir := t.TempDir()
	// Use a very short TTL so entries expire immediately
	cache := NewCache(cacheDir, "google_safe_browsing", 1*time.Nanosecond)

	result := &CheckResult{
		Level:    ThreatLevelDangerous,
		Provider: "Google Safe Browsing",
		Details:  []string{"MALWARE"},
	}

	err := cache.Put("https://malware.example.com", result)
	require.NoError(t, err)

	// Wait for TTL to expire
	time.Sleep(2 * time.Millisecond)

	cached, ok := cache.Get("https://malware.example.com")
	assert.False(t, ok)
	assert.Nil(t, cached)
}

func TestCache_SeparateProviders(t *testing.T) {
	cacheDir := t.TempDir()
	googleCache := NewCache(cacheDir, "google_safe_browsing", 24*time.Hour)
	vtCache := NewCache(cacheDir, "virustotal", 24*time.Hour)

	googleResult := &CheckResult{
		Level:    ThreatLevelSafe,
		Provider: "Google Safe Browsing",
	}
	vtResult := &CheckResult{
		Level:    ThreatLevelDangerous,
		Provider: "VirusTotal",
		Details:  []string{"3 engine(s) flagged as malicious"},
	}

	url := "https://example.com"

	require.NoError(t, googleCache.Put(url, googleResult))
	require.NoError(t, vtCache.Put(url, vtResult))

	// Each provider returns its own result
	cached, ok := googleCache.Get(url)
	require.True(t, ok)
	assert.Equal(t, ThreatLevelSafe, cached.Level)

	cached, ok = vtCache.Get(url)
	require.True(t, ok)
	assert.Equal(t, ThreatLevelDangerous, cached.Level)
	assert.Equal(t, []string{"3 engine(s) flagged as malicious"}, cached.Details)
}

func TestCache_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, "virustotal", 24*time.Hour)

	result := &CheckResult{Level: ThreatLevelSafe, Provider: "VirusTotal"}
	require.NoError(t, cache.Put("https://example.com", result))

	// Verify it's cached
	_, ok := cache.Get("https://example.com")
	require.True(t, ok)

	// Clear and verify gone
	require.NoError(t, cache.Clear())

	_, ok = cache.Get("https://example.com")
	assert.False(t, ok)
}

func TestClearAll(t *testing.T) {
	cacheDir := t.TempDir()
	googleCache := NewCache(cacheDir, "google_safe_browsing", 24*time.Hour)
	vtCache := NewCache(cacheDir, "virustotal", 24*time.Hour)

	result := &CheckResult{Level: ThreatLevelSafe, Provider: "test"}
	require.NoError(t, googleCache.Put("https://a.com", result))
	require.NoError(t, vtCache.Put("https://b.com", result))

	require.NoError(t, ClearAll(cacheDir))

	_, ok := googleCache.Get("https://a.com")
	assert.False(t, ok)

	_, ok = vtCache.Get("https://b.com")
	assert.False(t, ok)
}

func TestCache_PruneExpired(t *testing.T) {
	cacheDir := t.TempDir()
	// Use a very short TTL
	cache := NewCache(cacheDir, "google_safe_browsing", 1*time.Millisecond)

	result := &CheckResult{Level: ThreatLevelSafe, Provider: "Google Safe Browsing"}

	// Store multiple entries
	require.NoError(t, cache.Put("https://a.com", result))
	require.NoError(t, cache.Put("https://b.com", result))
	require.NoError(t, cache.Put("https://c.com", result))

	// Wait for TTL to expire
	time.Sleep(5 * time.Millisecond)

	// Add one fresh entry
	freshCache := NewCache(cacheDir, "google_safe_browsing", 24*time.Hour)
	require.NoError(t, freshCache.Put("https://fresh.com", result))

	// Prune with the short TTL — only expired entries should be removed
	cache.PruneExpired()

	// Expired entries should be gone
	_, ok := cache.Get("https://a.com")
	assert.False(t, ok)

	_, ok = cache.Get("https://b.com")
	assert.False(t, ok)

	_, ok = cache.Get("https://c.com")
	assert.False(t, ok)

	// Fresh entry should still exist (checked with a longer TTL cache)
	_, ok = freshCache.Get("https://fresh.com")
	assert.True(t, ok)
}
