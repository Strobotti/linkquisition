package safety

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	virusTotalBaseURL = "https://www.virustotal.com/api/v3"
)

type virusTotal struct {
	apiKey string
}

func newVirusTotal(apiKey string) *virusTotal {
	return &virusTotal{apiKey: apiKey}
}

func (v *virusTotal) Name() string {
	return "VirusTotal"
}

func (v *virusTotal) TestCredentials(ctx context.Context) error {
	url := virusTotalBaseURL + "/users/me"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-apikey", v.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

//nolint:cyclop
func (v *virusTotal) Check(ctx context.Context, targetURL string) (*CheckResult, error) {
	// VirusTotal URL lookup uses base64-encoded URL (without padding) as identifier
	urlID := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(targetURL))
	apiURL := virusTotalBaseURL + "/urls/" + urlID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-apikey", v.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 404 means URL hasn't been scanned yet — treat as safe (no data)
	if resp.StatusCode == http.StatusNotFound {
		return &CheckResult{
			Level:     ThreatLevelSafe,
			Provider:  v.Name(),
			Details:   []string{"URL not previously scanned"},
			CheckedAt: time.Now(),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result vtResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return v.interpretResult(&result), nil
}

func (v *virusTotal) interpretResult(result *vtResponse) *CheckResult {
	stats := result.Data.Attributes.LastAnalysisStats
	checkResult := &CheckResult{
		Level:     ThreatLevelSafe,
		Provider:  v.Name(),
		CheckedAt: time.Now(),
	}

	malicious := stats.Malicious + stats.Suspicious

	if malicious > 0 {
		checkResult.Level = ThreatLevelDangerous
		if stats.Malicious > 0 {
			checkResult.Details = append(checkResult.Details,
				fmt.Sprintf("%d engine(s) flagged as malicious", stats.Malicious))
		}
		if stats.Suspicious > 0 {
			checkResult.Details = append(checkResult.Details,
				fmt.Sprintf("%d engine(s) flagged as suspicious", stats.Suspicious))
		}
	}

	return checkResult
}

// VirusTotal API response types.
type vtResponse struct {
	Data vtData `json:"data"`
}

type vtData struct {
	Attributes vtAttributes `json:"attributes"`
}

type vtAttributes struct {
	LastAnalysisStats vtStats `json:"last_analysis_stats"`
}

type vtStats struct {
	Malicious  int `json:"malicious"`
	Suspicious int `json:"suspicious"`
	Harmless   int `json:"harmless"`
	Undetected int `json:"undetected"`
}
