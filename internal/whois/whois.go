// Package whois provides domain WHOIS information lookup and parsing.
package whois

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

// DomainInfo holds parsed WHOIS information for a domain.
type DomainInfo struct {
	Domain      string
	Registrar   string
	CreatedDate string
	ExpiryDate  string
	UpdatedDate string
	NameServers []string
	Status      []string
}

// Lookup performs a WHOIS query for the domain extracted from the given URL string.
// The context controls the timeout for the network operation.
// Lookup performs a WHOIS query for the domain extracted from the given URL string.
// The context controls the timeout for the network operation.
func Lookup(ctx context.Context, rawURL string) (*DomainInfo, error) {
	domain, err := extractDomain(rawURL)
	if err != nil {
		return nil, err
	}

	type result struct {
		raw string
		err error
	}

	ch := make(chan result, 1)

	go func() {
		raw, queryErr := whois.Whois(domain)
		ch <- result{raw: raw, err: queryErr}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("whois query timed out for %s: %w", domain, ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("whois query failed for %s: %w", domain, res.err)
		}

		parsed, parseErr := whoisparser.Parse(res.raw)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse whois response for %s: %w", domain, parseErr)
		}

		info := &DomainInfo{
			Domain:      parsed.Domain.Domain,
			Registrar:   parsed.Registrar.Name,
			CreatedDate: parsed.Domain.CreatedDate,
			ExpiryDate:  parsed.Domain.ExpirationDate,
			UpdatedDate: parsed.Domain.UpdatedDate,
			NameServers: parsed.Domain.NameServers,
			Status:      parsed.Domain.Status,
		}

		return info, nil
	}
}

// extractDomain parses a URL and returns the registrable domain (e.g. "example.com").
// For URLs without a scheme, it attempts to parse as-is.
func extractDomain(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL must not be empty")
	}

	// Ensure URL has a scheme for proper parsing
	u := rawURL
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("no host found in URL: %s", rawURL)
	}

	// Strip "www." prefix to get the registrable domain
	host = strings.TrimPrefix(host, "www.")

	return host, nil
}
