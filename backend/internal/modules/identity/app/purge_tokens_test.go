package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func loginInput(email string) app.LoginInput {
	return app.LoginInput{Email: email, Password: validPassword, ClientIP: "203.0.113.7"}
}

func TestPurgeExpiredRefreshTokens(t *testing.T) {
	t.Run("borra los tokens vencidos de todas las empresas y deja los vigentes", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "dueno@ferreteria.co", validPassword, domain.RoleOwner)
		_, calzado := e.seed(t, "dueno@calzado.co", validPassword, domain.RoleOwner)
		_, err := e.svc.Login(t.Context(), loginInput("dueno@ferreteria.co"))
		require.NoError(t, err)
		_, err = e.svc.Login(t.Context(), loginInput("dueno@calzado.co"))
		require.NoError(t, err)

		e.clock.now = e.clock.now.Add(domain.RefreshTokenTTL + time.Hour)
		_, err = e.svc.Login(t.Context(), loginInput("dueno@calzado.co"))
		require.NoError(t, err)

		purged, err := e.svc.PurgeExpiredRefreshTokens(t.Context())
		require.NoError(t, err)
		require.Equal(t, 2, purged)
		require.Len(t, e.tokens.byID, 1, "queda la sesión vigente")
		for _, tok := range e.tokens.byID {
			require.Equal(t, calzado.TenantID(), tok.TenantID())
		}
		e.span(t, "identity.PurgeExpiredRefreshTokens")
	})

	t.Run("sin tokens vencidos no borra nada", func(t *testing.T) {
		e := newEnv(t)
		purged, err := e.svc.PurgeExpiredRefreshTokens(t.Context())
		require.NoError(t, err)
		require.Zero(t, purged)
	})
}
