//go:build go1.20

package uri

import "errors"

func newErr(msg string) error {
	return errors.New(msg) //nolint:err113
}

// errorsJoin is a temporary indirection to keep support for go1.19
func errorsJoin(errs ...error) error {
	return errors.Join(errs...)
}
