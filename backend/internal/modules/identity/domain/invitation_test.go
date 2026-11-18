package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestInvitation(t *testing.T) {
	now := time.Date(2026, 11, 18, 15, 0, 0, 0, time.UTC)
	tenantID := uuid.Must(uuid.NewV7())
	ownerID := uuid.Must(uuid.NewV7())

	newInvitation := func(t *testing.T, role domain.Role) (*domain.Invitation, string) {
		t.Helper()
		inv, raw, err := domain.NewInvitation(tenantID, mustEmail(t, "Caja2@Negocio.co"), " Luis Pérez ", role, ownerID, now)
		require.NoError(t, err)
		return inv, raw
	}

	t.Run("crea una invitación pendiente cuyo enlace lleva el tenant y su hash", func(t *testing.T) {
		inv, raw := newInvitation(t, domain.RoleCashier)
		require.Equal(t, 7, int(inv.ID().Version()))
		require.Equal(t, "caja2@negocio.co", inv.Email().String())
		require.Equal(t, "Luis Pérez", inv.Name())
		require.Equal(t, domain.RoleCashier, inv.Role())
		require.Equal(t, ownerID, inv.InvitedBy())
		require.Equal(t, now.Add(domain.InvitationTTL), inv.ExpiresAt())
		require.Equal(t, domain.InvitationPending, inv.Status(now))

		gotTenant, gotHash, err := domain.ParseLinkToken(raw)
		require.NoError(t, err)
		require.Equal(t, tenantID, gotTenant)
		require.Equal(t, inv.Hash(), gotHash)
	})

	t.Run("rechaza invitaciones con datos inválidos", func(t *testing.T) {
		email := mustEmail(t, "bodega@negocio.co")
		tests := []struct {
			name    string
			email   domain.Email
			who     string
			role    domain.Role
			by      uuid.UUID
			wantErr error
		}{
			{name: "sin correo", email: domain.Email{}, who: "Luis", role: domain.RoleWarehouse, by: ownerID, wantErr: domain.ErrInvalidEmail},
			{name: "sin nombre", email: email, who: " ", role: domain.RoleWarehouse, by: ownerID, wantErr: domain.ErrInvalidUserName},
			{name: "con rol inválido", email: email, who: "Luis", role: domain.Role("root"), by: ownerID, wantErr: domain.ErrInvalidRole},
			{name: "sin quién invita", email: email, who: "Luis", role: domain.RoleWarehouse, by: uuid.Nil, wantErr: domain.ErrInvalidUserID},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, err := domain.NewInvitation(tenantID, tt.email, tt.who, tt.role, tt.by, now)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})

	t.Run("aceptar una invitación vigente la marca aceptada", func(t *testing.T) {
		inv, _ := newInvitation(t, domain.RoleAdmin)
		require.NoError(t, inv.Accept(now.Add(time.Hour)))
		require.Equal(t, domain.InvitationAccepted, inv.Status(now.Add(time.Hour)))
	})

	t.Run("la invitación vencida no se puede aceptar", func(t *testing.T) {
		inv, _ := newInvitation(t, domain.RoleCashier)
		later := now.Add(domain.InvitationTTL)
		require.Equal(t, domain.InvitationExpired, inv.Status(later))
		require.ErrorIs(t, inv.Accept(later), domain.ErrLinkExpired)
	})

	t.Run("la invitación aceptada no se acepta dos veces", func(t *testing.T) {
		inv, _ := newInvitation(t, domain.RoleCashier)
		require.NoError(t, inv.Accept(now))
		require.ErrorIs(t, inv.Accept(now), domain.ErrLinkUsed)
	})

	t.Run("la invitación revocada no se puede aceptar ni revocar otra vez", func(t *testing.T) {
		inv, _ := newInvitation(t, domain.RoleWarehouse)
		require.NoError(t, inv.Revoke(now))
		require.Equal(t, domain.InvitationRevoked, inv.Status(now))
		require.ErrorIs(t, inv.Accept(now), domain.ErrLinkUsed)
		require.ErrorIs(t, inv.Revoke(now), domain.ErrLinkUsed)
	})

	t.Run("se rehidrata con su estado", func(t *testing.T) {
		inv, _ := newInvitation(t, domain.RoleCashier)
		require.NoError(t, inv.Accept(now))
		got, err := domain.RehydrateInvitation(inv.Snapshot())
		require.NoError(t, err)
		require.Equal(t, domain.InvitationAccepted, got.Status(now))
		require.Equal(t, inv.Hash(), got.Hash())
	})
}

func TestParseLinkToken(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	for _, raw := range []string{"", "sin-punto", tenantID.String() + ".corto", "no-uuid.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		t.Run("rechaza el enlace "+raw, func(t *testing.T) {
			_, _, err := domain.ParseLinkToken(raw)
			require.ErrorIs(t, err, domain.ErrInvalidLink)
		})
	}
}
