//go:build integration

package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

const fakeHash = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA"

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 9, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.TenantRepository = (*postgres.TenantRepository)(nil)
	_ domain.UserRepository   = (*postgres.UserRepository)(nil)
)

type fixture struct {
	db      *db.DB
	tenants *postgres.TenantRepository
	users   *postgres.UserRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, tenants: postgres.NewTenantRepository(d), users: postgres.NewUserRepository(d)}
}

// createTenant stores a new tenant. The NIT is the same for every tenant: it
// is not unique.
func (f fixture) createTenant(t *testing.T) *domain.Tenant {
	t.Helper()
	nit, err := domain.ParseNIT("890903938-8")
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, now)
	require.NoError(t, err)
	require.NoError(t, f.tenants.Create(t.Context(), tenant))
	return tenant
}

func (f fixture) newUser(t *testing.T, tenantID uuid.UUID, email string) *domain.User {
	t.Helper()
	e, err := domain.ParseEmail(email)
	require.NoError(t, err)
	user, err := domain.NewUser(tenantID, e, "Ana Gómez", fakeHash, domain.RoleCashier, now)
	require.NoError(t, err)
	return user
}

func (f fixture) createUser(t *testing.T, tenantID uuid.UUID, email string) *domain.User {
	t.Helper()
	user := f.newUser(t, tenantID, email)
	require.NoError(t, f.users.Create(t.Context(), user))
	return user
}

func requireSameUser(t *testing.T, want, got *domain.User) {
	t.Helper()
	require.Equal(t, want.ID(), got.ID())
	require.Equal(t, want.TenantID(), got.TenantID())
	require.Equal(t, want.Email(), got.Email())
	require.Equal(t, want.Name(), got.Name())
	require.Equal(t, want.PasswordHash(), got.PasswordHash())
	require.Equal(t, want.Role(), got.Role())
	require.Equal(t, want.IsActive(), got.IsActive())
	require.Equal(t, want.CreatedAt(), got.CreatedAt())
}

// requireRLSViolation checks that err is "new row violates row-level security policy".
func requireRLSViolation(t *testing.T, err error) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "expected a PostgreSQL error, got %v", err)
	require.Equal(t, "42501", pgErr.Code, pgErr.Message)
}

func TestTenantRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("crea y lee un tenant", func(t *testing.T) {
		tenant := f.createTenant(t)

		got, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, tenant.ID(), got.ID())
		require.Equal(t, "Ferretería El Tornillo", got.Name())
		require.Equal(t, "890903938-8", got.NIT().String())
		require.Equal(t, domain.TenantActive, got.Status())
		require.Equal(t, now, got.CreatedAt())
	})

	t.Run("actualiza el estado de un tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		tenant.Suspend()
		require.NoError(t, f.tenants.Update(t.Context(), tenant))

		got, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, domain.TenantSuspended, got.Status())
	})

	t.Run("GetByID de un tenant inexistente devuelve ErrTenantNotFound", func(t *testing.T) {
		_, err := f.tenants.GetByID(t.Context(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrTenantNotFound)
	})

	t.Run("el tenant B no puede leer la fila del tenant A", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)

		var count int
		err := f.db.WithTenantTx(t.Context(), tenantB.ID(), func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM tenants WHERE id = $1", tenantA.ID()).Scan(&count)
		})
		require.NoError(t, err)
		require.Zero(t, count)
	})
}

func TestUserRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("crea y lee un usuario por id y por email", func(t *testing.T) {
		tenant := f.createTenant(t)
		user := f.createUser(t, tenant.ID(), "ana@ferreteria.co")

		byID, err := f.users.GetByID(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		requireSameUser(t, user, byID)

		byEmail, err := f.users.GetByEmail(t.Context(), tenant.ID(), user.Email())
		require.NoError(t, err)
		requireSameUser(t, user, byEmail)
	})

	t.Run("actualiza nombre, rol y estado de un usuario", func(t *testing.T) {
		tenant := f.createTenant(t)
		email, err := domain.ParseEmail("bodega@ferreteria.co")
		require.NoError(t, err)
		user, err := domain.NewUser(tenant.ID(), email, "Luis", fakeHash, domain.RoleCashier, now)
		require.NoError(t, err)
		require.NoError(t, f.users.Create(t.Context(), user))

		require.NoError(t, user.ChangeRole(domain.RoleWarehouse))
		user.Deactivate()
		require.NoError(t, f.users.Update(t.Context(), user))

		got, err := f.users.GetByID(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		requireSameUser(t, user, got)
	})

	t.Run("GetByID de un usuario inexistente devuelve ErrUserNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		_, err := f.users.GetByID(t.Context(), tenant.ID(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("email duplicado en el mismo tenant falla con ErrEmailTaken", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createUser(t, tenant.ID(), "ana@ferreteria.co")

		err := f.users.Create(t.Context(), f.newUser(t, tenant.ID(), "ANA@ferreteria.co"))
		require.ErrorIs(t, err, domain.ErrEmailTaken)
	})

	t.Run("el mismo email se permite en tenants distintos", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)
		f.createUser(t, tenantA.ID(), "ana@ferreteria.co")

		require.NoError(t, f.users.Create(t.Context(), f.newUser(t, tenantB.ID(), "ana@ferreteria.co")))
	})
}

func TestUserRepositoryTenantIsolation(t *testing.T) {
	f := newFixture(t)

	t.Run("un usuario del tenant A no aparece al consultar con el tenant B", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)
		user := f.createUser(t, tenantA.ID(), "ana@ferreteria.co")

		_, err := f.users.GetByID(t.Context(), tenantB.ID(), user.ID())
		require.ErrorIs(t, err, domain.ErrUserNotFound)

		_, err = f.users.GetByEmail(t.Context(), tenantB.ID(), user.Email())
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("un UPDATE desde el tenant B no afecta al usuario del tenant A", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)
		user := f.createUser(t, tenantA.ID(), "ana@ferreteria.co")

		// Bypasses the repository: it always scopes the transaction to the
		// user's own tenant, so only raw SQL can try the cross-tenant write.
		var affected int64
		err := f.db.WithTenantTx(t.Context(), tenantB.ID(), func(tx pgx.Tx) error {
			tag, err := tx.Exec(t.Context(), "UPDATE users SET name = 'Hackeado', active = false WHERE id = $1", user.ID())
			affected = tag.RowsAffected()
			return err
		})
		require.NoError(t, err)
		require.Zero(t, affected)

		got, err := f.users.GetByID(t.Context(), tenantA.ID(), user.ID())
		require.NoError(t, err)
		requireSameUser(t, user, got)
	})

	t.Run("Update de un usuario con un tenant ajeno devuelve ErrUserNotFound", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)
		user := f.createUser(t, tenantA.ID(), "ana@ferreteria.co")

		// Same id, but claiming to belong to tenant B.
		forged, err := domain.RehydrateUser(domain.UserSnapshot{
			ID: user.ID(), TenantID: tenantB.ID(), Email: user.Email(), Name: "Hackeado",
			PasswordHash: fakeHash, Role: domain.RoleOwner, Active: true, CreatedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.users.Update(t.Context(), forged), domain.ErrUserNotFound)

		got, err := f.users.GetByID(t.Context(), tenantA.ID(), user.ID())
		require.NoError(t, err)
		requireSameUser(t, user, got)
	})
}

// TestRowLevelSecurityPolicies exercises the policies with raw SQL as
// app_user, independently of the repositories.
func TestRowLevelSecurityPolicies(t *testing.T) {
	f := newFixture(t)

	t.Run("sin app.tenant_id no se ve ningún usuario ni tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createUser(t, tenant.ID(), "ana@ferreteria.co")

		var users, tenants int
		err := f.db.WithTx(t.Context(), func(tx pgx.Tx) error {
			if err := tx.QueryRow(t.Context(), "SELECT count(*) FROM users").Scan(&users); err != nil {
				return err
			}
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM tenants").Scan(&tenants)
		})
		require.NoError(t, err)
		require.Zero(t, users)
		require.Zero(t, tenants)
	})

	t.Run("insertar un usuario con tenant_id de otro tenant viola la política", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)

		err := f.db.WithTenantTx(t.Context(), tenantB.ID(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `
				INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at)
				VALUES ($1, $2, 'intruso@ferreteria.co', 'Intruso', $3, 'owner', true, now())`,
				uuid.Must(uuid.NewV7()), tenantA.ID(), fakeHash)
			return err
		})
		requireRLSViolation(t, err)
	})

	t.Run("crear un tenant distinto al de la transacción viola la política", func(t *testing.T) {
		tenant := f.createTenant(t)

		err := f.db.WithTenantTx(t.Context(), tenant.ID(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `
				INSERT INTO tenants (id, name, nit, status, created_at)
				VALUES ($1, 'Otra', '890903938-8', 'active', now())`, uuid.Must(uuid.NewV7()))
			return err
		})
		requireRLSViolation(t, err)
	})

	t.Run("un UPDATE no puede mover un usuario a otro tenant", func(t *testing.T) {
		tenantA := f.createTenant(t)
		tenantB := f.createTenant(t)
		user := f.createUser(t, tenantA.ID(), "ana@ferreteria.co")

		err := f.db.WithTenantTx(t.Context(), tenantA.ID(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "UPDATE users SET tenant_id = $1 WHERE id = $2", tenantB.ID(), user.ID())
			return err
		})
		requireRLSViolation(t, err)
	})
}
