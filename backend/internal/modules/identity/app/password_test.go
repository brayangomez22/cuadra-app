package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

const newPassword = "otra-clave-larga-9"

func TestPasswordReset(t *testing.T) {
	t.Run("el enlace cambia la contraseña una sola vez y cierra las sesiones", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@negocio.co", Password: validPassword})
		require.NoError(t, err)

		_, token, err := e.svc.IssuePasswordReset(t.Context(), s.adminActor, s.cashier.ID())
		require.NoError(t, err)
		require.Len(t, e.audit.of("user.password_reset"), 1)

		require.NoError(t, e.svc.ResetPassword(t.Context(), app.ResetPasswordInput{Token: token, Password: newPassword}))
		require.Zero(t, e.tokens.live(s.cashier.ID()))
		_, err = e.svc.Login(t.Context(), app.LoginInput{Email: "caja@negocio.co", Password: newPassword})
		require.NoError(t, err)

		err = e.svc.ResetPassword(t.Context(), app.ResetPasswordInput{Token: token, Password: validPassword})
		require.ErrorIs(t, err, domain.ErrLinkUsed)
		require.Equal(t, int64(1), e.counter(t, "cuadra.identity.password_resets", "result", "completed"))
	})

	t.Run("solo el último enlace emitido sirve", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, first, err := e.svc.IssuePasswordReset(t.Context(), s.ownerActor, s.cashier.ID())
		require.NoError(t, err)
		_, _, err = e.svc.IssuePasswordReset(t.Context(), s.ownerActor, s.cashier.ID())
		require.NoError(t, err)
		err = e.svc.ResetPassword(t.Context(), app.ResetPasswordInput{Token: first, Password: newPassword})
		require.ErrorIs(t, err, domain.ErrLinkUsed)
	})

	t.Run("un admin no restablece la contraseña del dueño", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, _, err := e.svc.IssuePasswordReset(t.Context(), s.adminActor, s.owner.ID())
		require.ErrorIs(t, err, domain.ErrOwnerOnly)
	})
}

func TestChangePassword(t *testing.T) {
	t.Run("cambia la propia contraseña con la actual", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		cashier := app.Actor{TenantID: s.tenant.ID(), UserID: s.cashier.ID()}
		require.NoError(t, e.svc.ChangePassword(t.Context(), cashier, app.PasswordChange{Current: validPassword, New: newPassword}))
		_, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@negocio.co", Password: newPassword})
		require.NoError(t, err)
	})

	t.Run("rechaza la contraseña actual equivocada y una nueva débil", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		cashier := app.Actor{TenantID: s.tenant.ID(), UserID: s.cashier.ID()}
		err := e.svc.ChangePassword(t.Context(), cashier, app.PasswordChange{Current: "no-es-esta-123", New: newPassword})
		require.ErrorIs(t, err, domain.ErrWrongPassword)
		err = e.svc.ChangePassword(t.Context(), cashier, app.PasswordChange{Current: validPassword, New: "corta"})
		require.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})
}

func TestLinkTokensStayOutOfSpans(t *testing.T) {
	e := newEnv(t)
	s := e.seedStaff(t)
	_, token, err := e.svc.IssuePasswordReset(t.Context(), s.ownerActor, s.cashier.ID())
	require.NoError(t, err)
	require.NoError(t, e.svc.ResetPassword(t.Context(), app.ResetPasswordInput{Token: token, Password: newPassword}))
	invite := e.invite(t, s.ownerActor, "nueva@negocio.co", "cashier")

	for _, span := range e.spans.Ended() {
		for _, attr := range span.Attributes() {
			require.NotContains(t, attr.Value.String(), token, "span %s", span.Name())
			require.NotContains(t, attr.Value.String(), invite, "span %s", span.Name())
		}
	}
}
