package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// requireUsableRefresh checks that raw is the current refresh token of userID.
func requireUsableRefresh(t *testing.T, e *env, raw string, userID uuid.UUID) {
	t.Helper()
	tenantID, hash, err := domain.ParseRefreshToken(raw)
	require.NoError(t, err)
	stored, err := e.tokens.GetByHash(t.Context(), tenantID, hash)
	require.NoError(t, err)
	require.Equal(t, userID, stored.UserID())
	require.False(t, stored.IsRevoked())
}

func TestRefresh(t *testing.T) {
	// loggedIn signs a cashier in and returns the session.
	loggedIn := func(t *testing.T, e *env) app.Session {
		t.Helper()
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@eltornillo.co", Password: validPassword, ClientIP: "203.0.113.7"})
		require.NoError(t, err)
		return session
	}
	familyOf := func(t *testing.T, e *env, raw string) uuid.UUID {
		t.Helper()
		tenantID, hash, err := domain.ParseRefreshToken(raw)
		require.NoError(t, err)
		stored, err := e.tokens.GetByHash(t.Context(), tenantID, hash)
		require.NoError(t, err)
		return stored.FamilyID()
	}

	t.Run("rota el token: el nuevo sirve y el anterior queda revocado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.clock.Advance(time.Hour)

		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.NoError(t, err)
		require.NotEqual(t, first.RefreshToken, second.RefreshToken)
		require.Equal(t, first.Profile.User.ID(), second.Profile.User.ID())
		require.Equal(t, e.clock.Now().Add(15*time.Minute), second.AccessExpiresAt)
		require.Equal(t, e.clock.Now().Add(domain.RefreshTokenTTL), second.RefreshExpiresAt)
		requireUsableRefresh(t, e, second.RefreshToken, first.Profile.User.ID())
		require.Equal(t, familyOf(t, e, first.RefreshToken), familyOf(t, e, second.RefreshToken))
		e.span(t, "identity.RefreshToken")
	})

	t.Run("un refresh token reutilizado invalida toda la familia", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)
		require.NoError(t, err)

		// The first token is presented again: someone else has a copy.
		_, err = e.svc.Refresh(t.Context(), first.RefreshToken)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked(), "every token of the family is revoked")
		}
		_, err = e.svc.Refresh(t.Context(), second.RefreshToken)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused, "the legitimate successor stops working too")
		require.EqualValues(t, 1, e.counter(t, "cuadra.identity.refresh_reuse_detected", "", ""),
			"counted once: when the reuse revoked live tokens")
	})

	t.Run("una rotación concurrente perdida se trata como reutilización", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.tokens.beforeRotate = func() {
			e.tokens.beforeRotate = nil
			_, err := e.svc.Refresh(t.Context(), first.RefreshToken)
			require.NoError(t, err)
		}

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)
		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked())
		}
	})

	t.Run("token vencido es rechazado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.clock.Advance(domain.RefreshTokenTTL)

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrRefreshTokenExpired)
	})

	t.Run("token mal formado o desconocido es rechazado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		tenantID, _, err := domain.ParseRefreshToken(first.RefreshToken)
		require.NoError(t, err)
		_, unknown, err := domain.NewRefreshToken(tenantID, uuid.Must(uuid.NewV7()), e.clock.Now())
		require.NoError(t, err)

		for _, raw := range []string{"", "basura", unknown} {
			_, err := e.svc.Refresh(t.Context(), raw)
			require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
		}
	})

	t.Run("usuario desactivado no puede refrescar y su familia queda revocada", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		user := first.Profile.User
		user.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), user))

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked())
		}
	})

	t.Run("tenant suspendido no puede refrescar", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		tenant := first.Profile.Tenant
		tenant.Suspend()
		require.NoError(t, e.tenants.Update(t.Context(), tenant))

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	})

	t.Run("el refresh refleja el rol actual del usuario", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		user := first.Profile.User
		require.NoError(t, user.ChangeRole(domain.RoleAdmin))
		require.NoError(t, e.users.Update(t.Context(), user))

		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.NoError(t, err)
		require.Contains(t, second.AccessToken, ":admin")
	})
}

func TestLogout(t *testing.T) {
	t.Run("revoca la familia y el token ya no sirve", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@eltornillo.co", Password: validPassword})
		require.NoError(t, err)

		require.NoError(t, e.svc.Logout(t.Context(), session.RefreshToken))

		_, err = e.svc.Refresh(t.Context(), session.RefreshToken)
		require.Error(t, err)
		require.Zero(t, e.counter(t, "cuadra.identity.refresh_reuse_detected", "", ""), "a logged-out token is not a theft signal")
		e.span(t, "identity.Logout")
	})

	t.Run("con un token desconocido o mal formado no falla", func(t *testing.T) {
		e := newEnv(t)
		_, unknown, err := domain.NewRefreshToken(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), e.clock.Now())
		require.NoError(t, err)

		for _, raw := range []string{"", "basura", unknown} {
			require.NoError(t, e.svc.Logout(t.Context(), raw))
		}
	})
}

func TestSessionAccess(t *testing.T) {
	t.Run("el inicio de sesión devuelve los módulos activos y los permisos del rol", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "caja@modaurbana.co", validPassword, domain.RoleCashier)
		e.modules.activate(tenant.ID(), "work_orders")

		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@modaurbana.co", Password: validPassword, ClientIP: "203.0.113.7"})

		require.NoError(t, err)
		require.Equal(t, []string{"work_orders"}, session.Profile.Modules)
		require.Contains(t, session.Profile.Permissions, domain.Permission("work_orders:write"))
		require.NotContains(t, session.Profile.Permissions, domain.PermCatalogWrite)
	})

	t.Run("el registro devuelve el owner con todos los permisos del core y sin módulos", func(t *testing.T) {
		e := newEnv(t)

		session, err := e.svc.SignUp(t.Context(), signUpInput())

		require.NoError(t, err)
		require.Empty(t, session.Profile.Modules)
		var core []domain.Permission
		for _, g := range domain.CoreGrants() {
			core = append(core, g.Permission)
		}
		require.Equal(t, core, session.Profile.Permissions)
	})
}
