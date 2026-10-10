package domain

import "errors"

// Domain errors of the identity module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidEmail         = errors.New("identity: invalid email")
	ErrInvalidNIT           = errors.New("identity: invalid NIT")
	ErrInvalidNITCheckDigit = errors.New("identity: NIT check digit does not match")
	ErrInvalidRole          = errors.New("identity: invalid role")
	ErrInvalidTenantName    = errors.New("identity: invalid tenant name")
	ErrInvalidTenantID      = errors.New("identity: invalid tenant id")
	ErrInvalidUserName      = errors.New("identity: invalid user name")
	ErrPasswordTooShort     = errors.New("identity: password too short")
	ErrPasswordTooLong      = errors.New("identity: password too long")
	ErrEmptyPasswordHash    = errors.New("identity: empty password hash")
)
