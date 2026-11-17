package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestSignUpRateLimit(t *testing.T) {
	t.Run("cuenta cada registro por la IP del cliente", func(t *testing.T) {
		e := newEnv(t)
		in := signUpInput()
		in.ClientIP = "198.51.100.7"

		_, err := e.svc.SignUp(t.Context(), in)
		require.NoError(t, err)
		require.Equal(t, []string{"198.51.100.7"}, e.signupLimiter.keys)
	})

	t.Run("pasado el límite responde con la espera y no crea nada", func(t *testing.T) {
		e := newEnv(t)
		e.signupLimiter.denied["198.51.100.7"] = true
		in := signUpInput()
		in.ClientIP = "198.51.100.7"

		_, err := e.svc.SignUp(t.Context(), in)
		require.ErrorIs(t, err, domain.ErrTooManyAttempts)
		var limited *app.RateLimitError
		require.ErrorAs(t, err, &limited)
		require.Positive(t, limited.RetryAfter)
		require.Empty(t, e.tenants.byID)
	})
}
