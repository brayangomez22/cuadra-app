// Package taxid holds the Colombian tax ID (NIT) and its DIAN check digit.
// It is a shared kernel: a pure value object that more than one module's
// domain validates (the tenant's NIT, a customer's, later the electronic
// invoice's). It depends only on the standard library.
package taxid

import "errors"

// Errors of ParseNIT. The modules' HTTP adapters map them to status codes.
var (
	ErrInvalidNIT           = errors.New("taxid: invalid NIT")
	ErrInvalidNITCheckDigit = errors.New("taxid: NIT check digit does not match")
)
