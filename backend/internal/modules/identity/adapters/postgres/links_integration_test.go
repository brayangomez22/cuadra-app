//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

var (
	_ domain.InvitationRepository    = (*postgres.InvitationRepository)(nil)
	_ domain.PasswordResetRepository = (*postgres.PasswordResetRepository)(nil)
)

func (f fixture) invite(t *testing.T, tenantID uuid.UUID, by *domain.User, email string) (*domain.Invitation, []byte) {
	t.Helper()
	inv, token, err := domain.NewInvitation(tenantID, mustEmail(t, email), "Luis", domain.RoleWarehouse, by.ID(), now)
	require.NoError(t, err)
	require.NoError(t, postgres.NewInvitationRepository(f.db).Create(t.Context(), inv))
	_, hash, err := domain.ParseLinkToken(token)
	require.NoError(t, err)
	return inv, hash
}

func TestInvitationRepository(t *testing.T) {
	f := newFixture(t)
	invitations := postgres.NewInvitationRepository(f.db)

	t.Run("guarda, busca por hash, acepta una sola vez y deja de listarla", func(t *testing.T) {
		tenant := f.createTenant(t)
		owner := f.createUser(t, tenant.ID(), "duena@ferreteria.co")
		inv, hash := f.invite(t, tenant.ID(), owner, "bodega@ferreteria.co")

		got, err := invitations.GetByHash(t.Context(), tenant.ID(), hash)
		require.NoError(t, err)
		require.Equal(t, inv.Snapshot(), got.Snapshot())
		open, err := invitations.ListOpen(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Len(t, open, 1)

		require.NoError(t, got.Accept(now.Add(time.Hour)))
		require.NoError(t, invitations.Close(t.Context(), got))
		require.ErrorIs(t, invitations.Close(t.Context(), got), domain.ErrLinkUsed)
		open, err = invitations.ListOpen(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Empty(t, open)
	})

	t.Run("invitar otra vez revoca las invitaciones abiertas del correo", func(t *testing.T) {
		tenant := f.createTenant(t)
		owner := f.createUser(t, tenant.ID(), "dueno@almacen.co")
		inv, _ := f.invite(t, tenant.ID(), owner, "caja@almacen.co")
		require.NoError(t, invitations.RevokeOpen(t.Context(), tenant.ID(), inv.Email(), now))
		got, err := invitations.GetByID(t.Context(), tenant.ID(), inv.ID())
		require.NoError(t, err)
		require.Equal(t, domain.InvitationRevoked, got.Status(now))
	})

	t.Run("las invitaciones de una empresa no se ven ni se cierran desde otra", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		owner := f.createUser(t, tenant.ID(), "duena@ferreteria.co")
		inv, hash := f.invite(t, tenant.ID(), owner, "bodega@ferreteria.co")

		_, err := invitations.GetByHash(t.Context(), other.ID(), hash)
		require.ErrorIs(t, err, domain.ErrLinkNotFound)
		_, err = invitations.GetByID(t.Context(), other.ID(), inv.ID())
		require.ErrorIs(t, err, domain.ErrInvitationNotFound)
		open, err := invitations.ListOpen(t.Context(), other.ID())
		require.NoError(t, err)
		require.Empty(t, open)

		snap := inv.Snapshot()
		snap.TenantID = other.ID()
		forged, err := domain.RehydrateInvitation(snap)
		require.NoError(t, err)
		require.NoError(t, forged.Revoke(now))
		require.ErrorIs(t, invitations.Close(t.Context(), forged), domain.ErrLinkUsed)
	})
}

func TestPasswordResetRepository(t *testing.T) {
	f := newFixture(t)
	resets := postgres.NewPasswordResetRepository(f.db)

	issue := func(t *testing.T, tenant *domain.Tenant, user *domain.User, at time.Time) (*domain.PasswordReset, []byte) {
		t.Helper()
		r, token, err := domain.NewPasswordReset(tenant.ID(), user.ID(), user.ID(), at)
		require.NoError(t, err)
		require.NoError(t, resets.Issue(t.Context(), r))
		_, hash, err := domain.ParseLinkToken(token)
		require.NoError(t, err)
		return r, hash
	}

	t.Run("se usa una sola vez y emitir otro gasta el anterior", func(t *testing.T) {
		tenant := f.createTenant(t)
		user := f.createUser(t, tenant.ID(), "caja@ferreteria.co")
		_, firstHash := issue(t, tenant, user, now)
		_, secondHash := issue(t, tenant, user, now.Add(time.Minute))

		first, err := resets.GetByHash(t.Context(), tenant.ID(), firstHash)
		require.NoError(t, err)
		require.ErrorIs(t, first.Use(now.Add(2*time.Minute)), domain.ErrLinkUsed)

		second, err := resets.GetByHash(t.Context(), tenant.ID(), secondHash)
		require.NoError(t, err)
		require.NoError(t, second.Use(now.Add(2*time.Minute)))
		require.NoError(t, resets.MarkUsed(t.Context(), second))
		require.ErrorIs(t, resets.MarkUsed(t.Context(), second), domain.ErrLinkUsed)
	})

	t.Run("el restablecimiento de una empresa no se ve desde otra", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		user := f.createUser(t, tenant.ID(), "caja@almacen.co")
		_, hash := issue(t, tenant, user, now)
		_, err := resets.GetByHash(t.Context(), other.ID(), hash)
		require.ErrorIs(t, err, domain.ErrLinkNotFound)
	})
}

func TestUserRepositoryAdministration(t *testing.T) {
	f := newFixture(t)
	tokens := postgres.NewRefreshTokenRepository(f.db)

	t.Run("lista los usuarios de la empresa, cuenta sus dueños activos y revoca sesiones", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		owner := f.newUser(t, tenant.ID(), "duena@ferreteria.co")
		require.NoError(t, owner.ChangeRole(domain.RoleOwner))
		require.NoError(t, f.users.Create(t.Context(), owner))
		cashier := f.createUser(t, tenant.ID(), "caja@ferreteria.co")
		f.createUser(t, other.ID(), "caja@almacen.co")

		users, err := f.users.List(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Len(t, users, 2)

		err = db.NewTransactor(f.db).InTx(t.Context(), tenant.ID(), func(ctx context.Context) error {
			owners, err := f.users.LockActiveOwners(ctx, tenant.ID())
			require.Equal(t, 1, owners)
			return err
		})
		require.NoError(t, err)

		token, _, err := domain.NewRefreshToken(tenant.ID(), cashier.ID(), now)
		require.NoError(t, err)
		require.NoError(t, tokens.Create(t.Context(), token))
		revoked, err := tokens.RevokeUser(t.Context(), other.ID(), cashier.ID(), now)
		require.NoError(t, err)
		require.Zero(t, revoked, "another tenant cannot end the user's sessions")
		revoked, err = tokens.RevokeUser(t.Context(), tenant.ID(), cashier.ID(), now)
		require.NoError(t, err)
		require.Equal(t, int64(1), revoked)
	})
}
