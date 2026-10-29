package domain

import (
	"errors"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

// Domain errors of the identity module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidEmail          = errors.New("identity: invalid email")
	ErrInvalidNIT            = taxid.ErrInvalidNIT
	ErrInvalidNITCheckDigit  = taxid.ErrInvalidNITCheckDigit
	ErrInvalidRole           = errors.New("identity: invalid role")
	ErrInvalidPermission     = errors.New("identity: invalid permission")
	ErrDuplicatePermission   = errors.New("identity: permission registered twice")
	ErrInvalidTenantName     = errors.New("identity: invalid tenant name")
	ErrInvalidTenantID       = errors.New("identity: invalid tenant id")
	ErrInvalidTenantStatus   = errors.New("identity: invalid tenant status")
	ErrInvalidDefaultTaxRate = errors.New("identity: invalid default tax rate")
	ErrInvalidUserID         = errors.New("identity: invalid user id")
	ErrInvalidUserName       = errors.New("identity: invalid user name")
	ErrPasswordTooShort      = errors.New("identity: password too short")
	ErrPasswordTooLong       = errors.New("identity: password too long")
	ErrEmptyPasswordHash     = errors.New("identity: empty password hash")
	ErrTenantNotFound        = errors.New("identity: tenant not found")
	// ErrNotDemoTenant reports a purge of a tenant that is not a demo one:
	// only demonstration tenants can be deleted.
	ErrNotDemoTenant = errors.New("identity: not a demo tenant")
	ErrUserNotFound  = errors.New("identity: user not found")
	ErrEmailTaken    = errors.New("identity: email already registered in the tenant")
	// ErrUnknownBusinessTemplate reports a business template that does not
	// exist or is not a well-formed ID.
	ErrUnknownBusinessTemplate = errors.New("identity: unknown business template")

	ErrInvalidCredentials      = errors.New("identity: invalid credentials")
	ErrTenantSuspended         = errors.New("identity: tenant suspended")
	ErrTenantSelectionRequired = errors.New("identity: credentials match several tenants")
	ErrTooManyAttempts         = errors.New("identity: too many login attempts")
	// ErrNotAuthorized means valid credentials of a user whose role may not
	// authorize the action.
	ErrNotAuthorized        = errors.New("identity: the user may not authorize the action")
	ErrInvalidRefreshToken  = errors.New("identity: invalid refresh token")
	ErrRefreshTokenNotFound = errors.New("identity: refresh token not found")
	ErrRefreshTokenExpired  = errors.New("identity: refresh token expired")
	ErrRefreshTokenReused   = errors.New("identity: refresh token reused")
	// ErrDemoUnavailable means there is no demo to enter: it is off, the
	// business template has none or it has no active user of the role.
	ErrDemoUnavailable = errors.New("identity: demo unavailable")
)
