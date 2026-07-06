package uri

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"unicode/utf8"
)

// IsIP indicates if the URI host was specified using an IP address (v4 or v6).
func (a authorityInfo) IsIP() bool {
	// IPvFuture won't parse as a netip.Addr
	return a.isIPv4 || (a.isIPv6 && !a.isIPvFuture)
}

// IPAddr returns the parsed netip.Addr whenever IsIP is true (or the zero value whenever false).
func (a authorityInfo) IPAddr() netip.Addr {
	if !a.IsIP() {
		return netip.Addr{}
	}

	unescapedHost, _ := url.PathUnescape(a.host) // TODO
	addr, _ := netip.ParseAddr(unescapedHost)

	return addr
}

//nolint:dupword,mnd // false positive in the BNF comment
//nolint:mnd // straightforward interpretation, no need to define a constant
func validateIPv4(host string) error {
	//
	// check for IPv4 address
	//
	// The host SHOULD check
	// the string syntactically for a dotted-decimal number before
	// looking it up in the Domain Name System.
	//
	// IPv4 may **NOT** contain percent-encoded escaped characters, e.g. 192.168.0.%31 is not valid.
	// Reference: https://www.rfc-editor.org/rfc/rfc3986#appendix-A
	//
	//  IPv4address   = dec-octet "." dec-octet "." dec-octet "." dec-octet
	//  dec-octet     = DIGIT                 ; 0-9
	//    / %x31-39 DIGIT         ; 10-99
	//    / "1" 2DIGIT            ; 100-199
	//    / "2" %x30-34 DIGIT     ; 200-249
	//    / "25" %x30-35          ; 250-255
	var (
		partNum, digitNum uint8
		currentPart       [3]byte
	)

	for i := 0; i < len(host); i++ {
		b := host[i]

		switch {
		case isDigit(b):
			switch digitNum {
			case 0:
				if b > '2' {
					return errValueGreater255
				}
			case 1:
				if currentPart[0] == '2' && b > '5' {
					return errValueGreater255
				}
			case 2:
				if currentPart[0] == '2' && currentPart[1] == '5' && b > '5' {
					return errValueGreater255
				}
			default:
				return errValueGreater255
			}

			currentPart[digitNum] = b
			digitNum++
		case b == '.':
			if digitNum == 0 {
				return errAtLeastOneDigit
			}
			if digitNum > 1 && currentPart[0] == '0' {
				return errLeadingZero
			}

			partNum++
			if partNum > 3 {
				return errTooLong
			}
			digitNum = 0
		default:
			// optimize the default path to return a value error
			return errInvalidCharacter
		}
	}

	if partNum < 3 {
		return errTooShort
	}

	return nil
}

func validateIPv6(host string) error {
	// * check IPv6
	// IPv6address =                            6( h16 ":" ) ls32
	//            /                       "::" 5( h16 ":" ) ls32
	//            / [               h16 ] "::" 4( h16 ":" ) ls32
	//            / [ *1( h16 ":" ) h16 ] "::" 3( h16 ":" ) ls32
	//            / [ *2( h16 ":" ) h16 ] "::" 2( h16 ":" ) ls32
	//            / [ *3( h16 ":" ) h16 ] "::"    h16 ":"   ls32
	//            / [ *4( h16 ":" ) h16 ] "::"              ls32
	//            / [ *5( h16 ":" ) h16 ] "::"              h16
	//            / [ *6( h16 ":" ) h16 ] "::"
	//
	// ls32        = ( h16 ":" h16 ) / IPv4address
	//             ; least-significant 32 bits of address
	//
	// h16         = 1*4HEXDIG
	//             ; 16 bits of address represented in hexadecimal
	// In addition to RFC 8986, RFC6874 states that IPv6 may include a zone ID like so:
	//  IPv6address *("%25" ZoneID}
	// The zone ID may be percent-encoded
	//  ZoneID = 1*( unreserved / pct-encoded )
	//
	var ipv6WithoutZone, zoneID string
	idx := strings.IndexRune(host, '%')
	switch {
	case idx == 0:
		return errorsJoin(
			ErrInvalidHostAddress,
			fmt.Errorf("only the zoneID of an IPv6 literal may be percent-encoded: %w", ErrURI),
		)
	case idx > 0:
		ipv6WithoutZone = host[:idx]
		zoneID = host[idx:]
	default:
		ipv6WithoutZone = host
	}

	// check for IPv6 address
	// IPv6 may contain a percent-encoded escaped zone identifier: unescape it first if present
	addr, err := netip.ParseAddr(ipv6WithoutZone)
	if err != nil {
		return errorsJoin(
			ErrInvalidHostAddress,
			fmt.Errorf("a square-bracketed host part should be a valid IPv6 address: %q: %w", host, ErrURI),
		)
	}

	if !addr.Is6() {
		// RFC3986 stipulates that only IPv6 addresses are within square brackets
		return errorsJoin(
			ErrInvalidHostAddress,
			fmt.Errorf("a square-bracketed host part should not contain an IPv4 address: %q: %w", host, ErrURI),
		)
	}

	if zoneID == "" {
		return nil
	}

	// check the escaping of the zoneID. Notice than if a zone is added, it must not be empty.
	if len(zoneID) < 4 || zoneID[1] != '2' || zoneID[2] != '5' {
		return errorsJoin(
			ErrInvalidHostAddress,
			fmt.Errorf("IPv6 zoneID separator in URI must be percent-encoded with %%25, but got: %q: %w", zoneID, ErrURI),
		)
	}

	if err := validateUnreservedWithExtra(zoneID, nil); err != nil {
		return errorsJoin(
			ErrInvalidHostAddress,
			fmt.Errorf("invalid IPv6 zoneID %q: %w", zoneID, err),
			ErrURI,
		)
	}

	return nil
}

// validateIPvFuture covers the special provision in the RFC for future IP scheme.
// The passed argument removes the heading "v" character.
//
// Example: http://[v6.fe80::a_en1]
//
//	Reference: https://www.rfc-editor.org/rfc/rfc3986#section-3.2.2
//
//	IPvFuture     = "v" 1*HEXDIG "." 1*( unreserved / sub-delims / ":" )
func validateIPvFuture(address string) error {
	var (
		offset                   int
		foundHexDigits, foundDot bool
	)

	for offset < len(address) {
		r, size := utf8.DecodeRuneInString(address[offset:])
		if r == utf8.RuneError {
			return fmt.Errorf("invalid UTF8 rune near: %q: %w", address[offset:], ErrURI)
		}
		offset += size

		if r == dotSeparator {
			foundDot = true

			break
		}

		if !isHex(r) {
			return fmt.Errorf(
				"invalid IP vFuture format: expect an hexadecimal version tag: %w", ErrURI,
			)
		}

		foundHexDigits = true
	}

	if !foundHexDigits || !foundDot {
		return fmt.Errorf(
			"invalid IP vFuture format: expect a '.' after an hexadecimal version tag: %w", ErrURI,
		)
	}

	if offset >= len(address) {
		return fmt.Errorf("invalid IP vFuture format: expect a non-empty address after the version tag: %w", ErrURI)
	}

	// TODO: wrong because IpvFuture is not escaped
	return validateUnreservedWithExtra(address[offset:], userInfoExtraRunes)
}
