//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

var (
	_ domain.RefreshTokenRepository = (*postgres.RefreshTokenRepository)(nil)
	_ domain.LoginDirectory         = (*postgres.LoginDirectory)(nil)
)

func newTenant(t *testing.T) *domain.Tenant {
	t.Helper()
	nit, err := domain.ParseNIT("890903938-8")
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, now)
	require.NoError(t, err)
	return tenant
}

func TestTenantRepositoryCreateWithOwner(t *testing.T) {
	f := newFixture(t)

	t.Run("crea el tenant y su dueño en una sola transacción", func(t *testing.T) {
		tenant := newTenant(t)
		owner := f.newUser(t, tenant.ID(), "dueno@eltornillo.co")

		require.NoError(t, f.tenants.CreateWithOwner(t.Context(), tenant, owner))

		_, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		got, err := f.users.GetByID(t.Context(), tenant.ID(), owner.ID())
		require.NoError(t, err)
		requireSameUser(t, owner, got)
	})

	t.Run("si el dueño no se puede guardar tampoco queda el tenant", func(t *testing.T) {
		tenant := newTenant(t)
		// An owner of another tenant violates the RLS policy of users.
		owner := f.newUser(t, uuid.Must(uuid.NewV7()), "dueno@eltornillo.co")

		err := f.tenants.CreateWithOwner(t.Context(), tenant, owner)

		requireRLSViolation(t, err)
		_, err = f.tenants.GetByID(t.Context(), tenant.ID())
		require.ErrorIs(t, err, domain.ErrTenantNotFound)
	})
}

type tokenFixture struct {
	fixture
	tokens *postgres.RefreshTokenRepository
}

func newTokenFixture(t *testing.T) tokenFixture {
	t.Helper()
	f := newFixture(t)
	return tokenFixture{fixture: f, tokens: postgres.NewRefreshTokenRepository(f.db)}
}

// createToken stores a new refresh token of a new user in tenant.
func (f tokenFixture) createToken(t *testing.T, tenant *domain.Tenant) *domain.RefreshToken {
	t.Helper()
	user := f.createUser(t, tenant.ID(), uuid.NewString()[:8]+"@eltornillo.co")
	token, _, err := domain.NewRefreshToken(tenant.ID(), user.ID(), now)
	require.NoError(t, err)
	require.NoError(t, f.tokens.Create(t.Context(), token))
	return token
}

func requireSameToken(t *testing.T, want, got *domain.RefreshToken) {
	t.Helper()
	require.Equal(t, want.ID(), got.ID())
	require.Equal(t, want.TenantID(), got.TenantID())
	require.Equal(t, want.UserID(), got.UserID())
	require.Equal(t, want.FamilyID(), got.FamilyID())
	require.Equal(t, want.Hash(), got.Hash())
	require.Equal(t, want.ExpiresAt(), got.ExpiresAt())
	require.Equal(t, want.CreatedAt(), got.CreatedAt())
	require.Equal(t, want.RevokedAt(), got.RevokedAt())
}

func TestRefreshTokenRepository(t *testing.T) {
	f := newTokenFixture(t)

	t.Run("crea y lee un token por su hash", func(t *testing.T) {
		tenant := f.createTenant(t)
		token := f.createToken(t, tenant)

		got, err := f.tokens.GetByHash(t.Context(), tenant.ID(), token.Hash())
		require.NoError(t, err)
		requireSameToken(t, token, got)
	})

	t.Run("un hash desconocido devuelve ErrRefreshTokenNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)

		_, err := f.tokens.GetByHash(t.Context(), tenant.ID(), make([]byte, 32))
		require.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)
	})

	t.Run("Rotate revoca el anterior y guarda el sucesor", func(t *testing.T) {
		tenant := f.createTenant(t)
		old := f.createToken(t, tenant)
		next, _, err := old.Rotate(now.Add(time.Hour))
		require.NoError(t, err)

		require.NoError(t, f.tokens.Rotate(t.Context(), old, next))

		gotOld, err := f.tokens.GetByHash(t.Context(), tenant.ID(), old.Hash())
		require.NoError(t, err)
		requireSameToken(t, old, gotOld)
		require.True(t, gotOld.IsRevoked())
		gotNext, err := f.tokens.GetByHash(t.Context(), tenant.ID(), next.Hash())
		require.NoError(t, err)
		requireSameToken(t, next, gotNext)
	})

	t.Run("Rotate de un token ya rotado devuelve ErrRefreshTokenReused y no guarda el sucesor", func(t *testing.T) {
		tenant := f.createTenant(t)
		stored := f.createToken(t, tenant)

		// Two requests read the same token; the first one rotates it.
		first, err := f.tokens.GetByHash(t.Context(), tenant.ID(), stored.Hash())
		require.NoError(t, err)
		second, err := f.tokens.GetByHash(t.Context(), tenant.ID(), stored.Hash())
		require.NoError(t, err)
		winner, _, err := first.Rotate(now)
		require.NoError(t, err)
		require.NoError(t, f.tokens.Rotate(t.Context(), first, winner))

		loser, _, err := second.Rotate(now)
		require.NoError(t, err)
		err = f.tokens.Rotate(t.Context(), second, loser)

		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)
		_, err = f.tokens.GetByHash(t.Context(), tenant.ID(), loser.Hash())
		require.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)
	})

	t.Run("RevokeFamily revoca solo los tokens vigentes de la familia y devuelve cuántos", func(t *testing.T) {
		tenant := f.createTenant(t)
		first := f.createToken(t, tenant)
		second, _, err := first.Rotate(now)
		require.NoError(t, err)
		require.NoError(t, f.tokens.Rotate(t.Context(), first, second))
		other := f.createToken(t, tenant)

		revoked, err := f.tokens.RevokeFamily(t.Context(), tenant.ID(), first.FamilyID(), now.Add(time.Minute))
		require.NoError(t, err)
		require.EqualValues(t, 1, revoked, "the rotated token was already revoked")

		got, err := f.tokens.GetByHash(t.Context(), tenant.ID(), second.Hash())
		require.NoError(t, err)
		require.True(t, got.IsRevoked())
		got, err = f.tokens.GetByHash(t.Context(), tenant.ID(), other.Hash())
		require.NoError(t, err)
		require.False(t, got.IsRevoked(), "another family is untouched")

		revoked, err = f.tokens.RevokeFamily(t.Context(), tenant.ID(), first.FamilyID(), now.Add(time.Minute))
		require.NoError(t, err)
		require.Zero(t, revoked)
	})
}

func TestRefreshTokenRepositoryTenantIsolation(t *testing.T) {
	f := newTokenFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)

	t.Run("un token del tenant A no se ve desde el tenant B", func(t *testing.T) {
		token := f.createToken(t, tenantA)

		_, err := f.tokens.GetByHash(t.Context(), tenantB.ID(), token.Hash())
		require.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)
	})

	t.Run("RevokeFamily desde el tenant B no revoca la familia del tenant A", func(t *testing.T) {
		token := f.createToken(t, tenantA)

		revoked, err := f.tokens.RevokeFamily(t.Context(), tenantB.ID(), token.FamilyID(), now)
		require.NoError(t, err)
		require.Zero(t, revoked)

		got, err := f.tokens.GetByHash(t.Context(), tenantA.ID(), token.Hash())
		require.NoError(t, err)
		require.False(t, got.IsRevoked())
	})

	t.Run("Rotate desde el tenant B no toca el token del tenant A", func(t *testing.T) {
		token := f.createToken(t, tenantA)
		forged, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
			ID: token.ID(), TenantID: tenantB.ID(), UserID: token.UserID(), FamilyID: token.FamilyID(),
			Hash: token.Hash(), ExpiresAt: token.ExpiresAt(), CreatedAt: token.CreatedAt(),
		})
		require.NoError(t, err)
		next, _, err := forged.Rotate(now)
		require.NoError(t, err)

		err = f.tokens.Rotate(t.Context(), forged, next)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

		got, err := f.tokens.GetByHash(t.Context(), tenantA.ID(), token.Hash())
		require.NoError(t, err)
		require.False(t, got.IsRevoked())
	})
}

func TestLoginDirectory(t *testing.T) {
	f := newFixture(t)
	directory := postgres.NewLoginDirectory(f.db)
	email := func(t *testing.T, s string) domain.Email {
		t.Helper()
		e, err := domain.ParseEmail(s)
		require.NoError(t, err)
		return e
	}

	t.Run("encuentra el email en todos los tenants donde está registrado", func(t *testing.T) {
		tenantA, tenantB, tenantC := f.createTenant(t), f.createTenant(t), f.createTenant(t)
		userA := f.createUser(t, tenantA.ID(), "compartido@gmail.com")
		userB := f.createUser(t, tenantB.ID(), "compartido@gmail.com")
		f.createUser(t, tenantC.ID(), "otro@gmail.com")

		got, err := directory.FindByEmail(t.Context(), email(t, "compartido@gmail.com"))

		require.NoError(t, err)
		require.ElementsMatch(t, []domain.LoginCandidate{
			{TenantID: tenantA.ID(), UserID: userA.ID()},
			{TenantID: tenantB.ID(), UserID: userB.ID()},
		}, got)
	})

	t.Run("un email no registrado no devuelve candidatos", func(t *testing.T) {
		got, err := directory.FindByEmail(t.Context(), email(t, "nadie@gmail.com"))
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("devuelve como máximo MaxLoginCandidates", func(t *testing.T) {
		for range domain.MaxLoginCandidates + 2 {
			f.createUser(t, f.createTenant(t).ID(), "muchos@gmail.com")
		}

		got, err := directory.FindByEmail(t.Context(), email(t, "muchos@gmail.com"))
		require.NoError(t, err)
		require.Len(t, got, domain.MaxLoginCandidates)
	})

	t.Run("app_user no puede asumir el rol auth_lookup para saltarse RLS", func(t *testing.T) {
		err := f.db.WithTx(t.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "SET LOCAL ROLE auth_lookup")
			return err
		})
		requireRLSViolation(t, err) // 42501: insufficient_privilege
	})

	t.Run("app_user sigue sin ver usuarios fuera de su tenant", func(t *testing.T) {
		tenantA := f.createTenant(t)
		f.createUser(t, tenantA.ID(), "aislado@gmail.com")

		var count int
		err := f.db.WithTenantTx(t.Context(), uuid.Must(uuid.NewV7()), func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM users WHERE email = 'aislado@gmail.com'").Scan(&count)
		})
		require.NoError(t, err)
		require.Zero(t, count)
	})
}
