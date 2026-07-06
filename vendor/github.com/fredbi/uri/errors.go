package uri

type (
	// Error from the github.com/fredbi/uri module.
	Error interface {
		error
	}
)

// Sentinel error
var ErrURI = Error(newErr("URI error"))

// Generic validation errors.
var (
	ErrInvalidURI       = Error(newErr("not a valid URI"))
	ErrInvalidCharacter = Error(newErr("invalid character in URI"))
	ErrInvalidEscaping  = Error(newErr("invalid percent-escaping sequence"))
)

// URI-specific errors
var (
	ErrNoSchemeFound   = Error(newErr("no scheme found in URI"))
	ErrInvalidScheme   = Error(newErr("invalid scheme in URI"))
	ErrInvalidQuery    = Error(newErr("invalid query string in URI"))
	ErrInvalidFragment = Error(newErr("invalid fragment in URI"))
)

// Authority-specific errors
var (
	ErrInvalidPath           = Error(newErr("invalid path in URI"))
	ErrInvalidHost           = Error(newErr("invalid host in URI"))
	ErrInvalidPort           = Error(newErr("invalid port in URI"))
	ErrInvalidUserInfo       = Error(newErr("invalid userinfo in URI"))
	ErrMissingHost           = Error(newErr("missing host in URI"))
	ErrInvalidHostAddress    = Error(newErr("invalid address for host"))
	ErrInvalidRegisteredName = Error(newErr("invalid host (registered name)"))
	ErrInvalidDNSName        = Error(newErr("invalid host (DNS name)"))
)

/*
// tells when a validation error originates from the authority part.
func isAuthorityErr(err error) bool {
	switch err {
	case ErrInvalidPath:
		return true
	case ErrInvalidHost:
		return true
	case ErrInvalidPort:
		return true
	case ErrMissingHost:
		return true
	case ErrInvalidHostAddress:
		return true
	case ErrInvalidRegisteredName:
		return true
	case ErrInvalidDNSName:
		return true
	default:
		log.Printf("error Is with: %q", spew.Sdump(err))
		return false
	}
}
*/

type ipError uint8

const (
	errInvalidCharacter ipError = iota
	errValueGreater255
	errAtLeastOneDigit
	errLeadingZero
	errTooLong
	errTooShort
)

func (e ipError) Error() string {
	switch e {
	case errInvalidCharacter:
		return "invalid character in IPv4 literal"
	case errValueGreater255:
		return "invalid IPv4 octet: IP field has value > 255"
	case errAtLeastOneDigit:
		return "IPv4 field must have at least one digit"
	case errLeadingZero:
		return "IPv4 field has octet with leading zero"
	case errTooLong:
		return "IPv4 address too long"
	case errTooShort:
		return "IPv4 address too short"
	default:
		return ""
	}
}

func (u uri) Err() error {
	return u.err
}

func (a authorityInfo) Err() error {
	return a.err
}
