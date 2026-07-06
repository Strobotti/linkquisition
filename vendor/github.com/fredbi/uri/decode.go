package uri

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validateUnreservedWithExtra(s string, acceptedRunes []rune) error {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			return errorsJoin(ErrInvalidEscaping,
				fmt.Errorf("invalid UTF8 rune near: %q: %w", s[i:], ErrURI),
			)
		}
		i += size

		// accepts percent-encoded sequences, but only if they correspond to a valid UTF-8 encoding
		if r == percentMark {
			if i >= len(s) {
				return errorsJoin(ErrInvalidEscaping,
					fmt.Errorf("incomplete escape sequence: %w", ErrURI),
				)
			}

			_, offset, err := unescapePercentEncoding(s[i:])
			if err != nil {
				return errorsJoin(ErrInvalidEscaping, err)
			}

			i += offset

			continue
		}

		// RFC grammar definitions:
		// sub-delims  = "!" / "$" / "&" / "'" / "(" / ")"
		//               / "*" / "+" / "," / ";" / "="
		// gen-delims  = ":" / "/" / "?" / "#" / "[" / "]" / "@"
		// unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
		// pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			// unreserved
			r != '-' && r != '.' && r != '_' && r != '~' &&
			// sub-delims
			r != '!' && r != '$' && r != '&' && r != '\'' && r != '(' && r != ')' &&
			r != '*' && r != '+' && r != ',' && r != ';' && r != '=' {
			runeFound := false
			for _, acceptedRune := range acceptedRunes {
				if r == acceptedRune {
					runeFound = true
					break
				}
			}

			if !runeFound {
				return fmt.Errorf("contains an invalid character: '%U' (%q) near %q: %w", r, r, s[i:], ErrURI)
			}
		}
	}

	return nil
}

func unescapePercentEncoding(s string) (rune, int, error) {
	var (
		offset          int
		codePoint       [utf8.UTFMax]byte
		codePointLength int
		err             error
	)

	if codePoint[0], err = unescapeSequence(s); err != nil {
		return utf8.RuneError, 0, err
	}

	codePointLength++
	offset += 2
	const (
		twoBytesUnicodePoint   = 0b11000000
		threeBytesUnicodePoint = 0b11100000
		fourBytesUnicodePoint  = 0b11110000
	)

	// escaped utf8 sequence
	if codePoint[0] >= twoBytesUnicodePoint {
		// expect another escaped sequence
		if offset >= len(s) {
			return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s, ErrURI)
		}

		if s[offset] != '%' {
			return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s[offset:], ErrURI)
		}
		offset++

		if codePoint[1], err = unescapeSequence(s[offset:]); err != nil {
			return utf8.RuneError, 0, err
		}

		codePointLength++
		offset += 2

		if codePoint[0] >= threeBytesUnicodePoint {
			// expect yet another escaped sequence
			if offset >= len(s) {
				return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s, ErrURI)
			}

			if s[offset] != '%' {
				return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s[offset:], ErrURI)
			}
			offset++

			if codePoint[2], err = unescapeSequence(s[offset:]); err != nil {
				return utf8.RuneError, 0, err
			}
			codePointLength++
			offset += 2

			if codePoint[0] >= fourBytesUnicodePoint {
				// expect a fourth escaped sequence
				if offset >= len(s) {
					return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s, ErrURI)
				}

				if s[offset] != '%' {
					return 0, 0, fmt.Errorf("expected a '%%' escape character, near: %q: %w", s[offset:], ErrURI)
				}
				offset++

				if codePoint[3], err = unescapeSequence(s[offset:]); err != nil {
					return utf8.RuneError, 0, err
				}
				codePointLength++
				offset += 2
			}
		}
	}

	unescapedRune, _ := utf8.DecodeRune(codePoint[:codePointLength])
	if unescapedRune == utf8.RuneError {
		return utf8.RuneError, 0, fmt.Errorf("the escaped code points do not add up to a valid rune: %w", ErrURI)
	}

	return unescapedRune, offset, nil
}

func unescapeSequence(escapeSequence string) (byte, error) {
	const (
		minEscapeSequenceLength = 2
	)
	if len(escapeSequence) < minEscapeSequenceLength {
		return 0, fmt.Errorf("expected escaping '%%' to be followed by 2 hex digits, near: %q: %w", escapeSequence, ErrURI)
	}

	if !isHex(escapeSequence[0]) || !isHex(escapeSequence[1]) {
		return 0, fmt.Errorf("part contains a malformed percent-encoded hex digit, near: %q: %w", escapeSequence, ErrURI)
	}

	return unhex(escapeSequence[0])<<4 | unhex(escapeSequence[1]), nil
}

func isHex[T byte | rune](c T) bool {
	switch {
	case isDigit(c):
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	default:
		return false
	}
}

func isNotDigit[T rune | byte](r T) bool {
	return r < '0' || r > '9'
}

func isDigit[T rune | byte](r T) bool {
	return r >= '0' && r <= '9'
}

func isASCIILetter[T byte | rune](c T) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func isNumerical(input string) bool {
	return strings.IndexFunc(input, isNotDigit[rune]) == -1
}

func unhex(c byte) byte {
	//nolint:mnd // there is no magic here: transforming a hex value in ASCII into its value
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
