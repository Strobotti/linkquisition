// Package uri is meant to be an RFC 3986 compliant URI builder and parser.
//
// This is based on the work from ttacon/uri (credits: Trey Tacon).
//
// This fork concentrates on RFC 3986 strictness for URI parsing and validation.
//
// Reference: https://tools.ietf.org/html/rfc3986
//
// Tests have been augmented with test suites of URI validators in other languages:
// perl, python, scala, .Net.
//
// Extra features like MySQL URIs present in the original repo have been removed.
package uri

import (
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// URI represents a general RFC3986 URI.
type URI interface {
	// Scheme the URI conforms to.
	Scheme() string

	// Authority information for the URI, including the "//" prefix.
	Authority() Authority

	// Query returns a map of key/value pairs of all parameters
	// in the query string of the URI.
	Query() url.Values

	// Fragment returns the fragment (component preceded by '#') in the
	// URI if there is one.
	Fragment() string

	// Builder returns a Builder that can be used to modify the URI.
	Builder() Builder

	// String representation of the URI
	String() string

	// Validate the different components of the URI
	Validate() error

	// Is the current port the default for this scheme?
	IsDefaultPort() bool
	// Default port for this scheme
	DefaultPort() int

	Err() error
}

// Authority information that a URI contains
// as specified by RFC3986.
//
// Username and password are given by UserInfo().
type Authority interface {
	UserInfo() string
	Host() string
	Port() string
	Path() string
	String() string
	Validate(...string) error

	IsIP() bool
	IPAddr() netip.Addr

	Err() error
}

type ipType struct {
	isIPv4      bool
	isIPv6      bool
	isIPvFuture bool
}

const (
	// char and string literals.
	colonMark          = ':'
	questionMark       = '?'
	fragmentMark       = '#'
	percentMark        = '%'
	atHost             = '@'
	slashMark          = '/'
	openingBracketMark = '['
	closingBracketMark = ']'
	dotSeparator       = '.'
	authorityPrefix    = "//"
)

const (
	// DNS name constants
	maxSegmentLength = 63
	maxDomainLength  = 255
)

var (
	// predefined sets of accecpted runes beyond the "unreserved" character set
	pcharExtraRunes           = []rune{colonMark, atHost} // pchar = unreserved | ':' | '@'
	queryOrFragmentExtraRunes = append(pcharExtraRunes, slashMark, questionMark)
	userInfoExtraRunes        = append(pcharExtraRunes, colonMark)
)

// IsURI tells if a URI is valid according to RFC3986/RFC397.
func IsURI(raw string) bool {
	_, err := Parse(raw)
	return err == nil
}

// IsURIReference tells if a URI reference is valid according to RFC3986/RFC397
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-4.1 and
// https://www.rfc-editor.org/rfc/rfc3986#section-4.2
func IsURIReference(raw string) bool {
	_, err := ParseReference(raw)
	return err == nil
}

// Parse attempts to parse a URI.
// It returns an error if the URI is not RFC3986-compliant.
func Parse(raw string) (URI, error) {
	return parse(raw, false)
}

// ParseReference attempts to parse a URI relative reference.
//
// It returns an error if the URI is not RFC3986-compliant.
func ParseReference(raw string) (URI, error) {
	return parse(raw, true)
}

func parse(raw string, withURIReference bool) (URI, error) {
	var (
		scheme string
		curr   int
	)

	schemeEnd := strings.IndexByte(raw, colonMark)      // position of a ":"
	hierPartEnd := strings.IndexByte(raw, questionMark) // position of a "?"
	queryEnd := strings.IndexByte(raw, fragmentMark)    // position of a "#"

	// exclude pathological input
	if schemeEnd == 0 || hierPartEnd == 0 || queryEnd == 0 {
		// ":", "?", "#"
		err := errorsJoin(
			ErrInvalidURI,
			fmt.Errorf("URI cannot start by a ':', '?' or '#' mark: %w", ErrURI),
		)
		return nil, err
	}

	if schemeEnd == 1 {
		return nil, errorsJoin(
			ErrInvalidScheme,
			fmt.Errorf("scheme has a minimum length of 2 characters: %w", ErrURI),
		)
	}

	if hierPartEnd == 1 || queryEnd == 1 {
		// ".:", ".?", ".#"
		err := errorsJoin(
			ErrInvalidURI,
			fmt.Errorf("invalid combination of start markers, near: %q: %w", raw[:2], ErrURI),
		)
		return nil, err
	}

	if hierPartEnd > 0 && hierPartEnd < schemeEnd || queryEnd > 0 && queryEnd < schemeEnd {
		// e.g. htt?p: ; h#ttp: ..
		mini, maxi := miniMaxi(hierPartEnd, schemeEnd, queryEnd, schemeEnd)
		err := errorsJoin(
			ErrInvalidURI,
			fmt.Errorf("URI part markers %q,%q,%q are in an incorrect order, near: %q: %w", colonMark, questionMark, fragmentMark, raw[mini:maxi], ErrURI),
		)
		return nil, err
	}

	if queryEnd > 0 && queryEnd < hierPartEnd {
		// e.g.  https://abc#a?b
		hierPartEnd = queryEnd
	}

	isRelative := strings.HasPrefix(raw, authorityPrefix)
	switch {
	case schemeEnd > 0 && !isRelative:
		scheme = raw[curr:schemeEnd]
		if schemeEnd+1 == len(raw) {
			// trailing ':' (e.g. http:)
			u := &uri{
				scheme: scheme,
			}

			return u, u.Validate()
		}
	case !withURIReference:
		// scheme is required for URI
		return nil, errorsJoin(
			ErrNoSchemeFound,
			fmt.Errorf("for URI (not URI reference), the scheme is required: %w", ErrURI),
		)
	case isRelative:
		// scheme is optional for URI references.
		//
		// start with // and a ':' is following... e.g //example.com:8080/path
		schemeEnd = -1
	}

	curr = schemeEnd + 1

	if hierPartEnd == len(raw)-1 || (hierPartEnd < 0 && queryEnd < 0) {
		// trailing ? or (no query & no fragment)
		if hierPartEnd < 0 {
			hierPartEnd = len(raw)
		}

		authority, err := parseAuthority(raw[curr:hierPartEnd])
		if err != nil {
			err = errorsJoin(ErrInvalidURI, err)
			return nil, err
		}

		u := &uri{
			scheme:    scheme,
			hierPart:  raw[curr:hierPartEnd],
			authority: authority,
		}

		return u, u.Validate()
	}

	var (
		hierPart, query, fragment string
		authority                 authorityInfo
		err                       error
	)

	if hierPartEnd > 0 {
		hierPart = raw[curr:hierPartEnd]
		authority, err = parseAuthority(hierPart)
		if err != nil {
			return nil, errorsJoin(ErrInvalidURI, err)
		}

		if hierPartEnd+1 < len(raw) {
			if queryEnd < 0 {
				// query ?, no fragment
				query = raw[hierPartEnd+1:]
			} else if hierPartEnd < queryEnd-1 {
				// query ?, fragment
				query = raw[hierPartEnd+1 : queryEnd]
			}
		}

		curr = hierPartEnd + 1
	}

	if queryEnd == len(raw)-1 && hierPartEnd < 0 {
		// trailing #,  no query "?"
		hierPart = raw[curr:queryEnd]
		authority, err = parseAuthority(hierPart)
		if err != nil {
			return nil, errorsJoin(ErrInvalidURI, err)
		}

		u := &uri{
			scheme:    scheme,
			hierPart:  hierPart,
			authority: authority,
			query:     query,
		}

		if err = u.Validate(); err != nil {
			return nil, err
		}

		return u, nil
	}

	if queryEnd > 0 {
		// there is a fragment
		if hierPartEnd < 0 {
			// no query
			hierPart = raw[curr:queryEnd]
			authority, err = parseAuthority(hierPart)
			if err != nil {
				return nil, errorsJoin(ErrInvalidURI, err)
			}
		}

		if queryEnd+1 < len(raw) {
			fragment = raw[queryEnd+1:]
		}
	}

	u := &uri{
		scheme:    scheme,
		hierPart:  hierPart,
		query:     query,
		fragment:  fragment,
		authority: authority,
	}

	return u, u.Validate()
}

type uri struct {
	// raw components
	scheme   string
	hierPart string
	query    string
	fragment string

	// parsed components
	authority authorityInfo
	err       error
}

func (u *uri) URI() URI {
	return u
}

// Scheme for this URI.
func (u *uri) Scheme() string {
	return u.scheme
}

// Authority information for the URI, including the "//" prefix.
func (u *uri) Authority() Authority {
	u.ensureAuthorityExists()
	return &u.authority
}

// Query returns a map of key/value pairs of all parameters
// in the query string of the URI.
//
//	This map contains the parsed query parameters like standard lib URL.Query().
func (u *uri) Query() url.Values {
	v, _ := url.ParseQuery(u.query)
	return v
}

func (u *uri) Fragment() string {
	return u.fragment
}

// Validate checks that all parts of a URI abide by allowed characters.
func (u *uri) Validate() error {
	if u.scheme != "" {
		if err := u.validateScheme(u.scheme); err != nil {
			u.err = err
			return err
		}
	}

	if u.query != "" {
		if err := u.validateQuery(u.query); err != nil {
			u.err = err
			return err
		}
	}

	if u.fragment != "" {
		if err := u.validateFragment(u.fragment); err != nil {
			u.err = err
			u.err = err
			return err
		}
	}

	if u.hierPart != "" {
		ip, err := u.authority.validate(u.scheme)
		if err != nil {
			u.err = err
			u.authority.err = err
			return err
		}
		u.authority.ipType = ip
	}

	// empty hierpart case
	return nil
}

// String representation of an URI.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-6.2.2.1 and later
func (u *uri) String() string {
	buf := strings.Builder{}
	buf.Grow(len(u.scheme) + 1 + len(u.query) + 1 + len(u.fragment) + 1 + u.authority.builderSize())

	if len(u.scheme) > 0 {
		buf.WriteString(u.scheme)
		buf.WriteByte(colonMark)
	}

	u.authority.buildString(&buf)

	if len(u.query) > 0 {
		buf.WriteByte(questionMark)
		buf.WriteString(u.query)
	}

	if len(u.fragment) > 0 {
		buf.WriteByte(fragmentMark)
		buf.WriteString(u.fragment)
	}

	return buf.String()
}

// validateScheme verifies the correctness of the scheme part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.1
// scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )
//
// NOTE: the scheme is not supposed to contain any percent-encoded sequence.
func (u *uri) validateScheme(scheme string) error {
	const minSchemeLength = 2
	if len(scheme) < minSchemeLength {
		return ErrInvalidScheme
	}

	c := scheme[0]
	if !isASCIILetter(c) {
		return errorsJoin(
			ErrInvalidScheme,
			fmt.Errorf("an URI scheme must start with an ASCII letter: %w", ErrURI),
		)
	}

	for i := 1; i < len(scheme); i++ {
		c := scheme[i]
		switch {
		case isDigit(c):
			// ok
		case isASCIILetter(c):
		// ok
		case c == '+' || c == '-' || c == '.':
		// ok
		default:
			return errorsJoin(
				ErrInvalidScheme,
				fmt.Errorf("invalid character %q found in scheme: %w", c, ErrURI),
			)
		}
	}

	return nil
}

// validateQuery validates the query part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.4
//
//	pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
//	query = *( pchar / "/" / "?" )
func (u *uri) validateQuery(query string) error {
	if err := validateUnreservedWithExtra(query, queryOrFragmentExtraRunes); err != nil {
		return errorsJoin(ErrInvalidQuery, err)
	}

	return nil
}

// validateFragment validatesthe fragment part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.5
//
//	pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
//
// fragment    = *( pchar / "/" / "?" )
func (u *uri) validateFragment(fragment string) error {
	if err := validateUnreservedWithExtra(fragment, queryOrFragmentExtraRunes); err != nil {
		return errorsJoin(ErrInvalidFragment, err)
	}

	return nil
}

type authorityInfo struct {
	ipType

	prefix   string
	userinfo string
	host     string
	port     string
	path     string
	err      error
}

func (a authorityInfo) UserInfo() string { return a.userinfo }
func (a authorityInfo) Host() string     { return a.host }
func (a authorityInfo) Port() string     { return a.port }
func (a authorityInfo) Path() string     { return a.path }
func (a authorityInfo) String() string {
	buf := strings.Builder{}
	buf.Grow(a.builderSize())
	a.buildString(&buf)

	return buf.String()
}

// Validate the Authority part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.2
func (a *authorityInfo) Validate(schemes ...string) error {
	ip, err := a.validate(schemes...)

	if err != nil {
		a.err = err

		return err
	}
	a.ipType = ip

	return nil
}

func (a authorityInfo) builderSize() int {
	return len(a.prefix) + len(a.userinfo) + 1 + len(a.host) + 2 + len(a.port) + 1 + len(a.path)
}

func (a authorityInfo) buildString(buf *strings.Builder) {
	buf.WriteString(a.prefix)
	buf.WriteString(a.userinfo)

	if len(a.userinfo) > 0 {
		buf.WriteByte(atHost)
	}

	if a.isIPv6 {
		buf.WriteString("[" + a.host + "]")
	} else {
		buf.WriteString(a.host)
	}

	if len(a.port) > 0 {
		buf.WriteByte(colonMark)
	}

	buf.WriteString(a.port)
	buf.WriteString(a.path)
}

func (a authorityInfo) validate(schemes ...string) (ipType, error) {
	var ip ipType

	if a.path != "" {
		if err := a.validatePath(a.path); err != nil {
			return ip, err
		}
	}

	if a.host != "" {
		var err error
		ip, err = a.validateHost(a.host, a.isIPv6, schemes...)
		if err != nil {
			return ip, err
		}
	}

	if a.port != "" {
		if err := a.validatePort(a.port, a.host); err != nil {
			return ip, err
		}
	}

	if a.userinfo != "" {
		if err := a.validateUserInfo(a.userinfo); err != nil {
			return ip, err
		}
	}

	return ip, nil
}

// validatePath validates the path part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.3
func (a authorityInfo) validatePath(path string) error {
	if a.host == "" && a.port == "" && len(path) >= 2 && path[0] == slashMark && path[1] == slashMark {
		return errorsJoin(
			ErrInvalidPath,
			fmt.Errorf(
				`if a URI does not contain an authority component, then the path cannot begin with two slash characters ("//"): %q: %w`,
				a.path, ErrURI,
			))
	}

	var previousPos int
	for pos, char := range path {
		if char != slashMark {
			continue
		}

		if pos > previousPos {
			if err := validateUnreservedWithExtra(path[previousPos:pos], pcharExtraRunes); err != nil {
				return errorsJoin(
					ErrInvalidPath,
					err,
				)
			}
		}

		previousPos = pos + 1
	}

	if previousPos < len(path) { // don't care if the last char was a separator
		if err := validateUnreservedWithExtra(path[previousPos:], pcharExtraRunes); err != nil {
			return errorsJoin(
				ErrInvalidPath,
				err,
			)
		}
	}

	return nil
}

// validateHost validates the host part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.2.2
func (a authorityInfo) validateHost(host string, isIPv6 bool, schemes ...string) (ipType, error) {
	// check for IP addresses
	// * IPv6 are required to be enclosed within '[]' (isIPv6=true), if an IPv6 zone is present,
	// there is a trailing escaped sequence, but the heading IPv6 literal must not be escaped.
	// * IPv4 are not percent-escaped: strict addresses never contain parts starting a zero (e.g. 012 should be 12).
	// * address the provision made in the RFC for a "IPvFuture"
	if isIPv6 {
		if host[0] == 'v' || host[0] == 'V' {
			if err := validateIPvFuture(host[1:]); err != nil {
				return ipType{}, errorsJoin(
					ErrInvalidHostAddress,
					err,
				)
			}

			return ipType{isIPv6: true, isIPvFuture: true}, nil
		}

		return ipType{isIPv6: true}, validateIPv6(host)
	}

	if err := validateIPv4(host); err == nil {
		return ipType{isIPv4: true}, nil
	}

	// This is not an IP: check for host DNS or registered name
	if err := validateHostForScheme(host, schemes...); err != nil {
		return ipType{}, errorsJoin(
			ErrInvalidHost,
			err,
		)
	}

	return ipType{}, nil
}

// validateHostForScheme validates the host according to 2 different sets of rules:
//   - if the scheme is a scheme well-known for using DNS host names, the DNS host validation applies (RFC)
//     (applies to schemes at: https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml)
//   - otherwise, applies the "registered-name" validation stated by RFC 3986:
//
// dns-name see: https://www.rfc-editor.org/rfc/rfc1034, https://www.rfc-editor.org/info/rfc5890
// reg-name    = *( unreserved / pct-encoded / sub-delims )
func validateHostForScheme(host string, schemes ...string) error {
	for _, scheme := range schemes {
		if UsesDNSHostValidation(scheme) {
			if err := validateDNSHostForScheme(host); err != nil {
				return err
			}
		}

		if err := validateRegisteredHostForScheme(host); err != nil {
			return err
		}
	}

	return nil
}

func validateRegisteredHostForScheme(host string) error {
	// RFC 3986 registered name
	if err := validateUnreservedWithExtra(host, nil); err != nil {
		return errorsJoin(
			ErrInvalidRegisteredName,
			err,
		)
	}

	return nil
}

// validatePort validates the port part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.2.3
//
// port = *DIGIT
func (a authorityInfo) validatePort(port, host string) error {
	const maxPort uint64 = 65535

	if !isNumerical(port) {
		return ErrInvalidPort
	}

	if host == "" {
		return errorsJoin(
			ErrMissingHost,
			fmt.Errorf("whenever a port is specified, a host part must be present: %w", ErrURI),
		)
	}

	portNum, _ := strconv.ParseUint(port, 10, 64)
	if portNum > maxPort {
		return errorsJoin(
			ErrInvalidPort,
			fmt.Errorf("a valid port lies in the range (0-%d): %w", maxPort, ErrURI),
		)
	}

	return nil
}

// validateUserInfo validates the userinfo part.
//
// Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.2.1
//
// userinfo    = *( unreserved / pct-encoded / sub-delims / ":" )
func (a authorityInfo) validateUserInfo(userinfo string) error {
	if err := validateUnreservedWithExtra(userinfo, userInfoExtraRunes); err != nil {
		return errorsJoin(
			ErrInvalidUserInfo,
			err,
		)
	}

	return nil
}

func parseAuthority(hier string) (authorityInfo, error) {
	// as per RFC 3986 Section 3.6
	var (
		prefix, userinfo, host, port, path string
		isIPv6                             bool
	)

	// authority sections MUST begin with a '//'
	if strings.HasPrefix(hier, authorityPrefix) {
		prefix = authorityPrefix
		hier = strings.TrimPrefix(hier, authorityPrefix)
	}

	if prefix == "" {
		path = hier
	} else {
		// authority   = [ userinfo "@" ] host [ ":" port ]
		slashEnd := strings.IndexByte(hier, slashMark)
		if slashEnd > -1 {
			if slashEnd < len(hier) {
				path = hier[slashEnd:]
			}
			hier = hier[:slashEnd]
		}

		host = hier
		if at := strings.IndexByte(host, atHost); at > 0 {
			userinfo = host[:at]
			if at+1 < len(host) {
				host = host[at+1:]
			}
		}

		if bracket := strings.IndexByte(host, openingBracketMark); bracket >= 0 {
			// ipv6 addresses: "["xx:yy:zz"]":port
			rawHost := host
			closingbracket := strings.IndexByte(host, closingBracketMark)
			switch {
			case closingbracket > bracket+1:
				host = host[bracket+1 : closingbracket]
				rawHost = rawHost[closingbracket+1:]
				isIPv6 = true
			case closingbracket > bracket:
				return authorityInfo{}, errorsJoin(
					ErrInvalidHostAddress,
					fmt.Errorf("empty IPv6 address: %w", ErrURI),
				)
			default:
				return authorityInfo{}, errorsJoin(
					ErrInvalidHostAddress,
					fmt.Errorf("mismatched square brackets: %w", ErrURI),
				)
			}

			if colon := strings.IndexByte(rawHost, colonMark); colon >= 0 {
				if colon+1 < len(rawHost) {
					port = rawHost[colon+1:]
				}
			}
		} else {
			if colon := strings.IndexByte(host, colonMark); colon >= 0 {
				if colon+1 < len(host) {
					port = host[colon+1:]
				}
				host = host[:colon]
			}
		}
	}

	return authorityInfo{
		prefix:   prefix,
		userinfo: userinfo,
		host:     host,
		port:     port,
		path:     path,
		ipType:   ipType{isIPv6: isIPv6},
	}, nil
}

func (u *uri) ensureAuthorityExists() {
	if u.authority.userinfo != "" ||
		u.authority.host != "" ||
		u.authority.port != "" {
		u.authority.prefix = authorityPrefix
	}
}

func miniMaxi(vals ...int) (int, int) {
	var mini, maxi int
	if len(vals) == 0 {
		return mini, maxi
	}

	mini, maxi = vals[0], vals[0]

	for _, val := range vals[1:] {
		if val < mini {
			mini = val
		}
		if val > maxi {
			maxi = val
		}
	}

	if mini < 0 {
		mini = 0
	}
	if maxi < 0 {
		maxi = 0
	}

	return mini, maxi
}
