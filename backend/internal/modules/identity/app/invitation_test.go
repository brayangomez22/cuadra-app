package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func (e *env) invite(t *testing.T, a app.Actor, email, role string) string {
	t.Helper()
	_, token, err := e.svc.InviteUser(t.Context(), a, app.InvitationInput{Email: email, Name: "Luis Pérez", Role: role})
	require.NoError(t, err)
	return token
}

func TestInviteUser(t *testing.T) {
	t.Run("invita con un enlace y queda en la auditoría sin el correo", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		inv, token, err := e.svc.InviteUser(t.Context(), s.adminActor,
			app.InvitationInput{Email: "Bodega@Negocio.co", Name: "Luis Pérez", Role: "warehouse"})
		require.NoError(t, err)
		require.NotEmpty(t, token)
		require.Equal(t, domain.InvitationPending, inv.Status(e.clock.Now()))

		open, err := e.svc.ListInvitations(t.Context(), s.adminActor)
		require.NoError(t, err)
		require.Len(t, open, 1)
		require.Equal(t, []app.AuditEntry{{
			Action: "user.invited", EntityID: inv.ID(), Details: map[string]string{"role": "warehouse"},
		}}, e.audit.of("user.invited"))
		require.Equal(t, int64(1), e.counter(t, "cuadra.identity.invitations", "result", "sent"))
	})

	t.Run("invitar con un correo ya registrado responde email_taken", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, _, err := e.svc.InviteUser(t.Context(), s.ownerActor,
			app.InvitationInput{Email: "caja@negocio.co", Name: "Otra", Role: "cashier"})
		require.ErrorIs(t, err, domain.ErrEmailTaken)
	})

	t.Run("un admin no invita dueños", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, _, err := e.svc.InviteUser(t.Context(), s.adminActor,
			app.InvitationInput{Email: "socio@negocio.co", Name: "Socio", Role: "owner"})
		require.ErrorIs(t, err, domain.ErrOwnerOnly)
	})

	t.Run("invitar otra vez el mismo correo revoca el enlace anterior", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		first := e.invite(t, s.ownerActor, "nuevo@negocio.co", "cashier")
		e.invite(t, s.ownerActor, "nuevo@negocio.co", "admin")

		_, err := e.svc.AcceptInvitation(t.Context(), app.AcceptInvitationInput{Token: first, Name: "Nuevo", Password: validPassword})
		require.ErrorIs(t, err, domain.ErrLinkUsed)
		open, err := e.svc.ListInvitations(t.Context(), s.ownerActor)
		require.NoError(t, err)
		require.Len(t, open, 1)
	})
}

func TestAcceptInvitation(t *testing.T) {
	t.Run("aceptar la invitación crea el usuario con el rol invitado y abre sesión", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		token := e.invite(t, s.adminActor, "bodega@negocio.co", "warehouse")

		view, err := e.svc.InvitationByToken(t.Context(), token)
		require.NoError(t, err)
		require.Equal(t, s.tenant.Name(), view.TenantName)
		require.Equal(t, "bodega@negocio.co", view.Invitation.Email().String())

		session, err := e.svc.AcceptInvitation(t.Context(), app.AcceptInvitationInput{Token: token, Name: " Luisa ", Password: validPassword})
		require.NoError(t, err)
		require.NotEmpty(t, session.AccessToken)
		require.Equal(t, domain.RoleWarehouse, session.Profile.User.Role())
		require.Equal(t, "Luisa", session.Profile.User.Name())
		require.Equal(t, int64(1), e.counter(t, "cuadra.identity.invitations", "result", "accepted"))

		_, err = e.svc.Login(t.Context(), app.LoginInput{Email: "bodega@negocio.co", Password: validPassword})
		require.NoError(t, err)
	})

	t.Run("el enlace aceptado no sirve dos veces", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		token := e.invite(t, s.adminActor, "bodega@negocio.co", "warehouse")
		in := app.AcceptInvitationInput{Token: token, Name: "Luisa", Password: validPassword}
		_, err := e.svc.AcceptInvitation(t.Context(), in)
		require.NoError(t, err)
		_, err = e.svc.AcceptInvitation(t.Context(), in)
		require.ErrorIs(t, err, domain.ErrLinkUsed)
	})

	t.Run("rechaza un enlace vencido, uno inventado y una contraseña débil", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		token := e.invite(t, s.adminActor, "bodega@negocio.co", "warehouse")

		_, err := e.svc.AcceptInvitation(t.Context(), app.AcceptInvitationInput{Token: token, Name: "Luisa", Password: "corta"})
		require.ErrorIs(t, err, domain.ErrPasswordTooShort)
		_, err = e.svc.AcceptInvitation(t.Context(), app.AcceptInvitationInput{Token: "inventado", Name: "Luisa", Password: validPassword})
		require.ErrorIs(t, err, domain.ErrLinkNotFound)

		e.clock.Advance(domain.InvitationTTL + time.Second)
		_, err = e.svc.AcceptInvitation(t.Context(), app.AcceptInvitationInput{Token: token, Name: "Luisa", Password: validPassword})
		require.ErrorIs(t, err, domain.ErrLinkExpired)
	})

	t.Run("revocar la invitación invalida su enlace", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		inv, token, err := e.svc.InviteUser(t.Context(), s.ownerActor,
			app.InvitationInput{Email: "caja2@negocio.co", Name: "Caja dos", Role: "cashier"})
		require.NoError(t, err)
		require.NoError(t, e.svc.RevokeInvitation(t.Context(), s.ownerActor, inv.ID()))
		_, err = e.svc.InvitationByToken(t.Context(), token)
		require.ErrorIs(t, err, domain.ErrLinkUsed)
	})
}
