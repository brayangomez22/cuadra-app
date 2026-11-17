package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer and audience of the access tokens: only this API issues and accepts them.
const (
	TokenIssuer   = "cuadra-api"
	TokenAudience = "cuadra-api"
)

// MinSecretLength is the minimum HS256 key size, in bytes (RFC 7518 §3.2).
const MinSecretLength = 32

var (
	// ErrInvalidToken means the token is malformed, forged, expired or not
	// meant for this API. The cause is wrapped for logs, never for clients.
	ErrInvalidToken = errors.New("auth: invalid token")
	// ErrWeakSecret is returned by NewJWT for a key shorter than MinSecretLength.
	ErrWeakSecret = fmt.Errorf("auth: JWT secret must have at least %d bytes", MinSecretLength)
)

// claims are the access token's claims: tid (tenant) and role besides the
// registered ones (sub is the user).
type claims struct {
	TenantID string `json:"tid"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWT issues and verifies HS256 access tokens.
type JWT struct {
	secret []byte
	// previous is the secret before a rotation: it only verifies, so the
	// tokens it signed keep working until they expire.
	previous []byte
	ttl      time.Duration
	now      func() time.Time
	parser   *jwt.Parser
}

// Option configures a JWT.
type Option func(*JWT)

// WithClock replaces time.Now (tests).
func WithClock(now func() time.Time) Option { return func(j *JWT) { j.now = now } }

// WithPreviousSecret keeps verifying the tokens signed with the secret
// before a rotation (JWT_SECRET_PREVIOUS). New tokens are signed with the
// current one only; once the previous tokens expire (the access token's
// ttl), the option can go.
func WithPreviousSecret(previous []byte) Option {
	return func(j *JWT) { j.previous = append([]byte(nil), previous...) }
}

// NewJWT returns a JWT signing with secret; tokens live for ttl.
func NewJWT(secret []byte, ttl time.Duration, opts ...Option) (*JWT, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrWeakSecret
	}
	j := &JWT{secret: append([]byte(nil), secret...), ttl: ttl, now: time.Now}
	for _, opt := range opts {
		opt(j)
	}
	if j.previous != nil && len(j.previous) < MinSecretLength {
		return nil, ErrWeakSecret
	}
	// Only HS256 is accepted: "none", other HMAC sizes and asymmetric
	// algorithms (key confusion) are rejected before the signature check.
	j.parser = jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return j.now() }),
	)
	return j, nil
}

// Issue returns an access token for the user and when it expires. It
// satisfies the identity module's AccessTokenIssuer port.
func (j *JWT) Issue(tenantID, userID uuid.UUID, role string) (string, time.Time, error) {
	now := j.now().UTC().Truncate(time.Second)
	expiresAt := now.Add(j.ttl)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		TenantID: tenantID.String(),
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign token: %w", err)
	}
	return token, expiresAt, nil
}

// keys are the secrets that verify a token: the current one and, during a
// rotation, the previous one.
func (j *JWT) keys(*jwt.Token) (any, error) {
	if j.previous == nil {
		return j.secret, nil
	}
	return jwt.VerificationKeySet{Keys: []jwt.VerificationKey{j.secret, j.previous}}, nil
}

// Verify checks the token's signature, algorithm, issuer, audience and
// lifetime, and returns its principal.
func (j *JWT) Verify(token string) (Principal, error) {
	var c claims
	if _, err := j.parser.ParseWithClaims(token, &c, j.keys); err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	tenantID, err := uuid.Parse(c.TenantID)
	if err != nil || tenantID == uuid.Nil {
		return Principal{}, fmt.Errorf("%w: bad tenant claim", ErrInvalidToken)
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil || userID == uuid.Nil {
		return Principal{}, fmt.Errorf("%w: bad subject claim", ErrInvalidToken)
	}
	if c.Role == "" {
		return Principal{}, fmt.Errorf("%w: missing role claim", ErrInvalidToken)
	}
	return Principal{TenantID: tenantID, UserID: userID, Role: c.Role}, nil
}
