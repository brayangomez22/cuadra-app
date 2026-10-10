package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var secret = []byte("0123456789abcdef0123456789abcdef-test-secret")

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func newJWT(t *testing.T) (*auth.JWT, *clock) {
	t.Helper()
	c := &clock{now: time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)}
	j, err := auth.NewJWT(secret, 15*time.Minute, auth.WithClock(c.Now))
	require.NoError(t, err)
	return j, c
}

// sign builds a token with arbitrary claims and method, as an attacker could.
func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)
	return token
}

func validClaims(c *clock, tenantID, userID uuid.UUID) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":  auth.TokenIssuer,
		"aud":  auth.TokenAudience,
		"sub":  userID.String(),
		"tid":  tenantID.String(),
		"role": "cashier",
		"iat":  c.now.Unix(),
		"exp":  c.now.Add(15 * time.Minute).Unix(),
	}
}

func TestJWT(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	t.Run("emite un token que se verifica con el mismo tenant, usuario y rol", func(t *testing.T) {
		j, c := newJWT(t)

		token, expiresAt, err := j.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)
		require.Equal(t, c.now.Add(15*time.Minute), expiresAt)

		p, err := j.Verify(token)
		require.NoError(t, err)
		require.Equal(t, auth.Principal{TenantID: tenantID, UserID: userID, Role: "cashier"}, p)
	})

	t.Run("el token vence a los 15 minutos", func(t *testing.T) {
		j, c := newJWT(t)
		token, _, err := j.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)

		c.now = c.now.Add(15*time.Minute - time.Second)
		_, err = j.Verify(token)
		require.NoError(t, err)

		c.now = c.now.Add(time.Second)
		_, err = j.Verify(token)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("rechaza tokens no válidos", func(t *testing.T) {
		j, c := newJWT(t)
		issued, _, err := j.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)

		// The payload says owner, but the signature is the cashier's.
		parts := strings.Split(issued, ".")
		forged := sign(t, jwt.SigningMethodHS256, []byte("otro-secreto-de-32-bytes-o-mas!!"), func() jwt.MapClaims {
			cl := validClaims(c, tenantID, userID)
			cl["role"] = "owner"
			return cl
		}())
		tampered := parts[0] + "." + strings.Split(forged, ".")[1] + "." + parts[2]

		withClaim := func(key string, value any) string {
			cl := validClaims(c, tenantID, userID)
			if value == nil {
				delete(cl, key)
			} else {
				cl[key] = value
			}
			return sign(t, jwt.SigningMethodHS256, secret, cl)
		}

		tests := []struct {
			name  string
			token string
		}{
			{name: "vacío", token: ""},
			{name: "basura", token: "no.es.jwt"},
			{name: "firmado con otro secreto", token: forged},
			{name: "payload alterado", token: tampered},
			{name: "alg none", token: sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims(c, tenantID, userID))},
			{name: "otro algoritmo HMAC (HS512)", token: sign(t, jwt.SigningMethodHS512, secret, validClaims(c, tenantID, userID))},
			{name: "emisor incorrecto", token: withClaim("iss", "otro")},
			{name: "audiencia incorrecta", token: withClaim("aud", "otro")},
			{name: "sin vencimiento", token: withClaim("exp", nil)},
			{name: "sin tenant", token: withClaim("tid", nil)},
			{name: "tenant que no es uuid", token: withClaim("tid", "ferreteria")},
			{name: "tenant nulo", token: withClaim("tid", uuid.Nil.String())},
			{name: "usuario que no es uuid", token: withClaim("sub", "ana")},
			{name: "sin rol", token: withClaim("role", nil)},
			{name: "vencido", token: withClaim("exp", c.now.Add(-time.Second).Unix())},
			{name: "emitido en el futuro", token: withClaim("iat", c.now.Add(time.Hour).Unix())},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := j.Verify(tt.token)
				require.ErrorIs(t, err, auth.ErrInvalidToken)
			})
		}
	})

	t.Run("rechaza un secreto de menos de 32 bytes", func(t *testing.T) {
		_, err := auth.NewJWT([]byte("corto"), 15*time.Minute)
		require.ErrorIs(t, err, auth.ErrWeakSecret)
	})
}
