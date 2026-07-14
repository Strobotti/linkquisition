//go:build !windows

package main

import (
	"context"
	"testing"

	"github.com/strobotti/linkquisition"
)

// testPlugin implements linkquisition.Plugin for testing.
type testPlugin struct {
	name   string
	result linkquisition.PluginResult
}

func (p *testPlugin) Metadata() linkquisition.PluginMetadata {
	return linkquisition.PluginMetadata{Name: p.name}
}

func (p *testPlugin) Setup(_ linkquisition.PluginServiceProvider, _ map[string]interface{}) error {
	return nil
}

func (p *testPlugin) ProcessURL(_ context.Context, url string) linkquisition.PluginResult {
	if p.result.URL == "" {
		p.result.URL = url
	}
	return p.result
}

func (p *testPlugin) Shutdown(_ context.Context) {}

func TestSingleLine_NoNewlines(t *testing.T) {
	input := "hello world"
	got := singleLine(input)
	if got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestSingleLine_WithNewlines(t *testing.T) {
	input := "line one\nline two\nline three"
	got := singleLine(input)
	expected := "line one line two line three"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestSingleLine_ConsecutiveNewlines(t *testing.T) {
	input := "hello\n\n\nworld"
	got := singleLine(input)
	// Multiple newlines should collapse to a single space
	expected := "hello world"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestSingleLine_Empty(t *testing.T) {
	got := singleLine("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSingleLine_OnlyNewlines(t *testing.T) {
	got := singleLine("\n\n\n")
	// Leading newlines with nothing before them produce nothing
	expected := ""
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestTracePlugins_NoPlugins(t *testing.T) {
	ctx := context.Background()
	result := tracePlugins(ctx, nil, "https://example.com")

	if result.url != "https://example.com" {
		t.Errorf("expected original URL, got %q", result.url)
	}
	if result.action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", result.action)
	}
}

func TestTracePlugins_SinglePluginModifiesURL(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "strip-tracking",
			result: linkquisition.PluginResult{
				URL:    "https://example.com/clean",
				Action: linkquisition.ActionContinue,
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://example.com/dirty?utm_source=test")

	if result.url != "https://example.com/clean" {
		t.Errorf("expected modified URL, got %q", result.url)
	}
	if result.action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", result.action)
	}
}

func TestTracePlugins_PluginBlocks(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "blocker",
			result: linkquisition.PluginResult{
				URL:     "https://malware.com",
				Action:  linkquisition.ActionBlock,
				Message: "Blocked: known malware site",
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://malware.com")

	if result.action != linkquisition.ActionBlock {
		t.Errorf("expected ActionBlock, got %v", result.action)
	}
	if result.message != "Blocked: known malware site" {
		t.Errorf("unexpected message: %q", result.message)
	}
}

func TestTracePlugins_PluginWarns(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "warner",
			result: linkquisition.PluginResult{
				URL:     "https://suspicious.com",
				Action:  linkquisition.ActionWarn,
				Message: "This URL looks suspicious",
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://suspicious.com")

	if result.action != linkquisition.ActionWarn {
		t.Errorf("expected ActionWarn, got %v", result.action)
	}
	if result.message != "This URL looks suspicious" {
		t.Errorf("unexpected message: %q", result.message)
	}
	if result.url != "https://suspicious.com" {
		t.Errorf("expected URL to be passed through, got %q", result.url)
	}
}

func TestTracePlugins_PluginOpenDirect(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "direct",
			result: linkquisition.PluginResult{
				URL:    "https://internal.corp.com",
				Action: linkquisition.ActionOpenDirect,
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://internal.corp.com")

	if result.action != linkquisition.ActionOpenDirect {
		t.Errorf("expected ActionOpenDirect, got %v", result.action)
	}
}

func TestTracePlugins_ChainStopsOnBlock(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "blocker",
			result: linkquisition.PluginResult{
				URL:     "https://evil.com",
				Action:  linkquisition.ActionBlock,
				Message: "blocked",
			},
		},
		&testPlugin{
			name: "should-not-run",
			result: linkquisition.PluginResult{
				URL:    "https://modified.com",
				Action: linkquisition.ActionContinue,
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://evil.com")

	if result.action != linkquisition.ActionBlock {
		t.Errorf("expected ActionBlock, got %v", result.action)
	}
	// The second plugin should not have modified the URL
	if result.url != "https://evil.com" {
		t.Errorf("expected original URL (second plugin should not run), got %q", result.url)
	}
}

func TestTracePlugins_ChainContinuesWithContinueChain(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "warn-but-continue",
			result: linkquisition.PluginResult{
				URL:           "https://example.com",
				Action:        linkquisition.ActionWarn,
				Message:       "heads up",
				ContinueChain: true,
			},
		},
		&testPlugin{
			name: "modifier",
			result: linkquisition.PluginResult{
				URL:    "https://example.com/modified",
				Action: linkquisition.ActionContinue,
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://example.com")

	// The warn action with ContinueChain=true should have let the second plugin run
	// But the tracePlugins function stores the first non-continue action as final
	if result.action != linkquisition.ActionWarn {
		t.Errorf("expected ActionWarn (from first plugin), got %v", result.action)
	}
}

func TestTracePlugins_MultiplePluginsModifyURL(t *testing.T) {
	plugins := []linkquisition.Plugin{
		&testPlugin{
			name: "step1",
			result: linkquisition.PluginResult{
				URL:    "https://example.com/step1",
				Action: linkquisition.ActionContinue,
			},
		},
		&testPlugin{
			name: "step2",
			result: linkquisition.PluginResult{
				URL:    "https://example.com/step2",
				Action: linkquisition.ActionContinue,
			},
		},
	}

	ctx := context.Background()
	result := tracePlugins(ctx, plugins, "https://example.com/original")

	if result.url != "https://example.com/step2" {
		t.Errorf("expected URL after both plugins, got %q", result.url)
	}
	if result.action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", result.action)
	}
}

func TestPrintBrowserMatch_BlockedSkips(t *testing.T) {
	settings := &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{Name: "Firefox", Command: "firefox %u", Matches: []linkquisition.BrowserMatch{
				{Type: "site", Value: "example.com"},
			}},
		},
	}

	result := testURLResult{
		url:    "https://example.com",
		action: linkquisition.ActionBlock,
	}

	// Should not panic — blocked URLs skip browser matching
	printBrowserMatch(settings, result)
	t.Log("ok")
}

func TestPrintBrowserMatch_OpenDirectSkips(t *testing.T) {
	settings := &linkquisition.Settings{}

	result := testURLResult{
		url:    "https://example.com",
		action: linkquisition.ActionOpenDirect,
	}

	// Should not panic — OpenDirect skips browser matching
	printBrowserMatch(settings, result)
	t.Log("ok")
}

func TestPrintBrowserMatch_MatchFound(t *testing.T) {
	settings := &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox %u",
				Matches: []linkquisition.BrowserMatch{
					{Type: "site", Value: "www.example.com"},
				},
			},
		},
	}

	result := testURLResult{
		url:    "https://www.example.com/page",
		action: linkquisition.ActionContinue,
	}

	// Should not panic — exercises the match-found path
	printBrowserMatch(settings, result)
	t.Log("ok")
}

func TestPrintBrowserMatch_NoMatch(t *testing.T) {
	settings := &linkquisition.Settings{
		Browsers: []linkquisition.BrowserSettings{
			{
				Name:    "Firefox",
				Command: "firefox %u",
				Hidden:  false,
				Matches: []linkquisition.BrowserMatch{
					{Type: "site", Value: "other.com"},
				},
			},
		},
	}

	result := testURLResult{
		url:    "https://nomatch.example.com/page",
		action: linkquisition.ActionContinue,
	}

	// Should not panic — exercises the no-match path (shows available browsers)
	printBrowserMatch(settings, result)
	t.Log("ok")
}

func TestPrintOutcome_Continue_NoChange(t *testing.T) {
	result := testURLResult{
		url:    "https://example.com",
		action: linkquisition.ActionContinue,
	}

	// Should not panic
	printOutcome(result, "https://example.com")
	t.Log("ok")
}

func TestPrintOutcome_Continue_URLChanged(t *testing.T) {
	result := testURLResult{
		url:    "https://example.com/clean",
		action: linkquisition.ActionContinue,
	}

	// Should not panic — prints "Final URL:"
	printOutcome(result, "https://example.com/dirty")
	t.Log("ok")
}

func TestPrintOutcome_Block(t *testing.T) {
	result := testURLResult{
		url:     "",
		action:  linkquisition.ActionBlock,
		message: "Blocked by policy",
	}
	printOutcome(result, "https://blocked.com")
	t.Log("ok")
}

func TestPrintOutcome_Warn(t *testing.T) {
	result := testURLResult{
		url:     "https://suspicious.com",
		action:  linkquisition.ActionWarn,
		message: "Suspicious URL detected",
	}
	printOutcome(result, "https://suspicious.com")
	t.Log("ok")
}

func TestPrintOutcome_OpenDirect(t *testing.T) {
	result := testURLResult{
		url:    "https://direct.com",
		action: linkquisition.ActionOpenDirect,
	}
	printOutcome(result, "https://direct.com")
	t.Log("ok")
}

func TestPrintPluginStep_Continue_Unchanged(t *testing.T) {
	r := linkquisition.PluginResult{
		URL:    "https://example.com",
		Action: linkquisition.ActionContinue,
	}
	// Should not panic
	printPluginStep(1, "sanitize", r, "https://example.com")
	t.Log("ok")
}

func TestPrintPluginStep_Continue_Changed(t *testing.T) {
	r := linkquisition.PluginResult{
		URL:    "https://example.com/clean",
		Action: linkquisition.ActionContinue,
	}
	printPluginStep(1, "sanitize", r, "https://example.com/dirty")
	t.Log("ok")
}

func TestPrintPluginStep_Block(t *testing.T) {
	r := linkquisition.PluginResult{
		URL:     "https://blocked.com",
		Action:  linkquisition.ActionBlock,
		Message: "Blocked\nMulti-line message",
	}
	printPluginStep(1, "defang", r, "https://blocked.com")
	t.Log("ok")
}

func TestPrintPluginStep_Warn(t *testing.T) {
	r := linkquisition.PluginResult{
		URL:     "https://warned.com",
		Action:  linkquisition.ActionWarn,
		Message: "Be careful",
	}
	printPluginStep(1, "shenanigans", r, "https://warned.com")
	t.Log("ok")
}

func TestPrintPluginStep_OpenDirect(t *testing.T) {
	r := linkquisition.PluginResult{
		URL:    "https://direct.com",
		Action: linkquisition.ActionOpenDirect,
	}
	printPluginStep(1, "terminus", r, "https://direct.com")
	t.Log("ok")
}
