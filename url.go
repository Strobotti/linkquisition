package linkquisition

import (
	"net/url"
	"regexp"

	"golang.org/x/net/publicsuffix"
)

type URL struct {
	url string
}

func NewURL(u string) *URL {
	return &URL{url: u}
}

func (u URL) GetDomain() (string, error) {
	parsedUrl, err := url.Parse(u.url)
	if err != nil {
		return "", err
	}

	// If the hostname is an IP address, we return it as is
	re := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
	if re.MatchString(parsedUrl.Hostname()) {
		return parsedUrl.Hostname(), nil
	}

	tldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(parsedUrl.Hostname())
	if err != nil {
		return "", err
	}

	return tldPlusOne, nil
}

func (u URL) GetSite() (string, error) {
	re := regexp.MustCompile(`^https?://([^/]+)(/|$)`)
	match := re.FindStringSubmatch(u.url)
	if len(match) > 1 {
		return match[1], nil
	}

	return "", nil
}
