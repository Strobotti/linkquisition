package safety

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	safeBrowsingLookupURL = "https://safebrowsing.googleapis.com/v4/threatMatches:find"
	safeBrowsingTestURL   = "https://safebrowsing.googleapis.com/v4/threatLists"
	safeBrowsingReportURL = "https://transparencyreport.google.com/safe-browsing/search?url="

	threatTypeMalware            = "MALWARE"
	threatTypeSocialEngineering  = "SOCIAL_ENGINEERING"
	threatTypeUnwantedSoftware   = "UNWANTED_SOFTWARE"
	threatTypePotentiallyHarmful = "POTENTIALLY_HARMFUL_APPLICATION"
)

type googleSafeBrowsing struct {
	apiKey string
}

func newGoogleSafeBrowsing(apiKey string) *googleSafeBrowsing {
	return &googleSafeBrowsing{apiKey: apiKey}
}

func (g *googleSafeBrowsing) Name() string {
	return ProviderNameGoogleSafeBrowsing
}

func (g *googleSafeBrowsing) TestCredentials(ctx context.Context) error {
	apiURL := safeBrowsingTestURL + "?key=" + g.apiKey

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

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
func (g *googleSafeBrowsing) Check(ctx context.Context, targetURL string) (*CheckResult, error) {
	reportURL := safeBrowsingReportURL + url.QueryEscape(targetURL)

	payload := safeBrowsingRequest{
		Client: sbClient{
			ClientID:      "linkquisition",
			ClientVersion: "1.0.0",
		},
		ThreatInfo: sbThreatInfo{
			ThreatTypes:      []string{threatTypeMalware, threatTypeSocialEngineering, threatTypeUnwantedSoftware, threatTypePotentiallyHarmful},
			PlatformTypes:    []string{"ANY_PLATFORM"},
			ThreatEntryTypes: []string{"URL"},
			ThreatEntries:    []sbThreatEntry{{URL: targetURL}},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := safeBrowsingLookupURL + "?key=" + g.apiKey

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result safeBrowsingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	checkResult := &CheckResult{
		Level:     ThreatLevelSafe,
		Provider:  g.Name(),
		ReportURL: reportURL,
		CheckedAt: time.Now(),
	}

	if len(result.Matches) > 0 {
		checkResult.Level = ThreatLevelDangerous
		for _, match := range result.Matches {
			checkResult.Details = append(checkResult.Details, match.ThreatType)
		}
	}

	return checkResult, nil
}

// Safe Browsing API request/response types.
type safeBrowsingRequest struct {
	Client     sbClient     `json:"client"`
	ThreatInfo sbThreatInfo `json:"threatInfo"`
}

type sbClient struct {
	ClientID      string `json:"clientId"`
	ClientVersion string `json:"clientVersion"`
}

type sbThreatInfo struct {
	ThreatTypes      []string        `json:"threatTypes"`
	PlatformTypes    []string        `json:"platformTypes"`
	ThreatEntryTypes []string        `json:"threatEntryTypes"`
	ThreatEntries    []sbThreatEntry `json:"threatEntries"`
}

type sbThreatEntry struct {
	URL string `json:"url"`
}

type safeBrowsingResponse struct {
	Matches []sbMatch `json:"matches"`
}

type sbMatch struct {
	ThreatType string `json:"threatType"`
}
