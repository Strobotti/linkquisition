package uri

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// UsesDNSHostValidation returns true if the provided scheme has host validation
// that does not follow RFC3986 (which is quite generic), and assumes a valid
// DNS hostname instead.
//
// This function is declared as a global variable that may be overridden at the package level,
// in case you need specific schemes to validate the host as a DNS name.
//
// See: https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
var UsesDNSHostValidation = func(scheme string) bool {
	switch scheme {
	// prioritize early exit on most commonly used schemes
	case "https", "http":
		return true
	case "file":
		return false
		// less commonly used schemes
	case "aaa":
		return true
	case "aaas":
		return true
	case "acap":
		return true
	case "acct":
		return true
	case "cap":
		return true
	case "cid":
		return true
	case "coap", "coaps", "coap+tcp", "coap+ws", "coaps+tcp", "coaps+ws":
		return true
	case "dav":
		return true
	case "dict":
		return true
	case "dns":
		return true
	case "dntp":
		return true
	case "finger":
		return true
	case "ftp":
		return true
	case "git":
		return true
	case "gopher":
		return true
	case "h323":
		return true
	case "iax":
		return true
	case "icap":
		return true
	case "im":
		return true
	case "imap":
		return true
	case "ipp", "ipps":
		return true
	case "irc", "irc6", "ircs":
		return true
	case "jms":
		return true
	case "ldap":
		return true
	case "mailto":
		return true
	case "mid":
		return true
	case "msrp", "msrps":
		return true
	case "nfs":
		return true
	case "nntp":
		return true
	case "ntp":
		return true
	case "postgresql":
		return true
	case "radius":
		return true
	case "redis":
		return true
	case "rmi":
		return true
	case "rtsp", "rtsps", "rtspu":
		return true
	case "rsync":
		return true
	case "sftp":
		return true
	case "skype":
		return true
	case "smtp":
		return true
	case "snmp":
		return true
	case "soap":
		return true
	case "ssh":
		return true
	case "steam":
		return true
	case "svn":
		return true
	case "tcp":
		return true
	case "telnet":
		return true
	case "udp":
		return true
	case "vnc":
		return true
	case "wais":
		return true
	case "ws":
		return true
	case "wss":
		return true
	}

	return false
}

func validateDNSHostForScheme(host string) error {
	// ref: https://datatracker.ietf.org/doc/html/rfc1035
	//	   <domain> ::= <subdomain> | " "
	//	   <subdomain> ::= <label> | <subdomain> "." <label>
	//	   <label> ::= <letter> [ [ <ldh-str> ] <let-dig> ]
	//     <ldh-str> ::= <let-dig-hyp> | <let-dig-hyp> <ldh-str>
	//	   <let-dig-hyp> ::= <let-dig> | "-"
	//	   <let-dig> ::= <letter> | <digit>
	//	   <letter> ::= any one of the 52 alphabetic characters A through Z in
	//	   upper case and a through z in lower case
	//	   <digit> ::= any one of the ten digits 0 through 9
	if len(host) > maxDomainLength {
		// The size considered is in bytes.
		// As a result, different escaping/normalization schemes
		// may or may not be valid for the same host.
		return errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("hostname is longer than the allowed 255 bytes: %w", ErrURI),
		)
	}
	if len(host) == 0 {
		return errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("a DNS name should not contain an empty segment: %w", ErrURI),
		)
	}

	for offset := 0; offset < len(host); {
		last, consumed, err := validateHostSegment(host[offset:])
		if err != nil {
			return err
		}

		if last != dotSeparator {
			break
		}

		offset += consumed
	}

	return nil
}

func validateHostSegment(s string) (rune, int, error) {
	// NOTE: this validator supports percent-encoded "." separators.
	last, offset, err := validateFirstRuneInSegment(s)
	if err != nil {
		return utf8.RuneError, 0, err
	}

	var (
		once          bool
		unescapedRune rune
	)

	for offset < len(s) {
		r, size := utf8.DecodeRuneInString(s[offset:])
		if r == utf8.RuneError {
			return utf8.RuneError, 0, errorsJoin(
				ErrInvalidDNSName,
				fmt.Errorf("invalid UTF8 rune near: %q: %w", s, ErrURI),
			)
		}
		once = true
		offset += size

		if r == percentMark {
			if offset >= len(s) {
				return utf8.RuneError, 0, errorsJoin(
					ErrInvalidDNSName,
					errorsJoin(ErrInvalidEscaping,
						fmt.Errorf("incomplete escape sequence: %w", ErrURI),
					))
			}

			unescapedRune, size, err = unescapePercentEncoding(s[offset:])
			if err != nil {
				return utf8.RuneError, 0, errorsJoin(
					ErrInvalidDNSName,
					errorsJoin(ErrInvalidEscaping, err),
				)
			}

			r = unescapedRune
			offset += size
		}

		if r == dotSeparator {
			// end of segment, possibly with an escaped "."
			if offset >= len(s) {
				return utf8.RuneError, 0, errorsJoin(
					ErrInvalidDNSName,
					fmt.Errorf("a DNS name should not contain an empty segment: %w", ErrURI),
				)
			}
			if !unicode.IsLetter(last) && !unicode.IsDigit(last) {
				return utf8.RuneError, 0, errorsJoin(
					ErrInvalidDNSName,
					fmt.Errorf("a segment in a DNS name must end with a letter or a digit: %q ends with %q: %w", s, last, ErrURI),
				)
			}

			return r, offset, nil
		}

		if offset > maxSegmentLength {
			return utf8.RuneError, 0, errorsJoin(
				ErrInvalidDNSName,
				fmt.Errorf("a segment in a DNS name should not be longer than 63 bytes: %q: %w", s[:offset], ErrURI),
			)
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return utf8.RuneError, 0, errorsJoin(
				ErrInvalidDNSName,
				fmt.Errorf("a segment in a DNS name must contain only letters, digits or '-': %q contains %q: %w", s, r, ErrURI),
			)
		}

		last = r
	}

	// last rune in segment
	if once && !unicode.IsLetter(last) && !unicode.IsDigit(last) {
		return utf8.RuneError, 0, errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("a segment in a DNS name must end with a letter or a digit: %q ends with %q: %w", s, last, ErrURI),
		)
	}

	return last, offset, nil
}

func validateFirstRuneInSegment(s string) (rune, int, error) {
	// validate the first rune for a DNS host segment
	var offset int
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return utf8.RuneError, 0, errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("invalid UTF8 rune near: %q: %w", s, ErrURI),
		)
	}
	if r == dotSeparator {
		return utf8.RuneError, 0, errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("a DNS name should not contain an empty segment: %w", ErrURI),
		)
	}
	offset += size

	if r == percentMark {
		if offset >= len(s) {
			return utf8.RuneError, 0, errorsJoin(
				errorsJoin(ErrInvalidEscaping,
					fmt.Errorf("incomplete escape sequence: %w", ErrURI),
				))
		}
		unescapedRune, consumed, e := unescapePercentEncoding(s[offset:])
		if e != nil {
			return utf8.RuneError, 0, errorsJoin(
				ErrInvalidDNSName,
				errorsJoin(ErrInvalidEscaping, e),
			)
		}

		r = unescapedRune
		offset += consumed
	}

	// If it is a number we fail here to fall back to IP parsing
	if _, err := strconv.Atoi(string([]rune{r}) + s[offset:]); err == nil {
		return utf8.RuneError, 0, errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("hostname cannot just be a number: %w", ErrURI))
	}

	if !unicode.IsLetter(r) && (!unicode.IsDigit(r) || offset >= len(s)) {
		return utf8.RuneError, 0, errorsJoin(
			ErrInvalidDNSName,
			fmt.Errorf("a segment in a DNS name must begin with a letter: %q starts with %q: %w", s, r, ErrURI),
		)
	}

	return r, offset, nil
}
