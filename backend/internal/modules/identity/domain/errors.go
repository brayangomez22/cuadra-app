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
	ErrInvalidTenantStatus  = errors.New("identity: invalid tenant status")
	ErrInvalidUserID        = errors.New("identity: invalid user id")
	ErrInvalidUserName      = errors.New("identity: invalid user name")
	ErrPasswordTooShort     = errors.New("identity: password too short")
	ErrPasswordTooLong      = errors.New("identity: password too long")
	ErrEmptyPasswordHash    = errors.New("identity: empty password hash")
	ErrTenantNotFound       = errors.New("identity: tenant not found")
	ErrUserNotFound         = errors.New("identity: user not found")
	ErrEmailTaken           = errors.New("identity: email already registered in the tenant")

	ErrInvalidCredentials      = errors.New("identity: invalid credentials")
	ErrTenantSuspended         = errors.New("identity: tenant suspended")
	ErrTenantSelectionRequired = errors.New("identity: credentials match several tenants")
	ErrTooManyAttempts         = errors.New("identity: too many login attempts")
	ErrInvalidRefreshToken     = errors.New("identity: invalid refresh token")
	ErrRefreshTokenNotFound    = errors.New("identity: refresh token not found")
	ErrRefreshTokenExpired     = errors.New("identity: refresh token expired")
	ErrRefreshTokenReused      = errors.New("identity: refresh token reused")
)
