//go:build integration

package postgres_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/postgres"
)

var now = time.Date(2026, 10, 16, 15, 0, 0, 0, time.UTC)

// createTenant stores a tenant row directly: tenant_modules references it.
func createTenant(t *testing.T, d *db.DB) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, d.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2)", id, now)
		return err
	}))
	return id
}

// set applies one change, unconditionally.
func set(t *testing.T, store *postgres.Store, tenantID uuid.UUID, module string, active bool) {
	t.Helper()
	require.NoError(t, store.Update(t.Context(), tenantID, func([]string) (*modules.Change, error) {
		return &modules.Change{Module: module, Active: active, By: uuid.Must(uuid.NewV7()), At: now}, nil
	}))
}

func TestStore(t *testing.T) {
	d := dbtest.New(t)
	store := postgres.NewStore(d)

	t.Run("activa y desactiva módulos", func(t *testing.T) {
		tenant := createTenant(t, d)

		set(t, store, tenant, "variants", true)
		set(t, store, tenant, "lots_expiry", true)
		active, err := store.Active(t.Context(), tenant)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"variants", "lots_expiry"}, active)

		set(t, store, tenant, "variants", false)
		active, err = store.Active(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"lots_expiry"}, active)
	})

	t.Run("activar dos veces o desactivar un módulo inactivo no falla", func(t *testing.T) {
		tenant := createTenant(t, d)

		set(t, store, tenant, "work_orders", true)
		set(t, store, tenant, "work_orders", true)
		set(t, store, tenant, "variants", false)

		active, err := store.Active(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"work_orders"}, active)
	})

	t.Run("decide recibe los módulos activos y sin cambio no escribe", func(t *testing.T) {
		tenant := createTenant(t, d)
		set(t, store, tenant, "variants", true)

		var seen []string
		require.NoError(t, store.Update(t.Context(), tenant, func(active []string) (*modules.Change, error) {
			seen = active
			return nil, nil
		}))

		require.Equal(t, []string{"variants"}, seen)
	})

	t.Run("un error de decide no escribe", func(t *testing.T) {
		tenant := createTenant(t, d)

		err := store.Update(t.Context(), tenant, func([]string) (*modules.Change, error) {
			return nil, modules.ErrDependencyInactive
		})

		require.ErrorIs(t, err, modules.ErrDependencyInactive)
		active, err := store.Active(t.Context(), tenant)
		require.NoError(t, err)
		require.Empty(t, active)
	})

	t.Run("serializa los cambios de un mismo tenant", func(t *testing.T) {
		tenant := createTenant(t, d)
		// Each change activates the module named after how many are active:
		// run one at a time, they activate m0…m9; interleaved, two would read
		// the same count and activate the same module.
		const writers = 10
		var wg sync.WaitGroup
		errs := make(chan error, writers)
		for range writers {
			wg.Go(func() {
				errs <- store.Update(t.Context(), tenant, func(active []string) (*modules.Change, error) {
					return &modules.Change{Module: "m" + string(rune('0'+len(active))), Active: true, By: uuid.Must(uuid.NewV7()), At: now}, nil
				})
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		active, err := store.Active(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, active, writers)
	})
}

func TestStoreTenantIsolation(t *testing.T) {
	d := dbtest.New(t)
	store := postgres.NewStore(d)
	tenantA, tenantB := createTenant(t, d), createTenant(t, d)
	set(t, store, tenantA, "variants", true)

	t.Run("el tenant B no ve los módulos del tenant A", func(t *testing.T) {
		active, err := store.Active(t.Context(), tenantB)
		require.NoError(t, err)
		require.Empty(t, active)
	})

	t.Run("desactivar desde el tenant B no cambia el tenant A", func(t *testing.T) {
		set(t, store, tenantB, "variants", false)

		active, err := store.Active(t.Context(), tenantA)
		require.NoError(t, err)
		require.Equal(t, []string{"variants"}, active)
	})

	t.Run("activar desde el tenant B solo lo activa en B", func(t *testing.T) {
		set(t, store, tenantB, "work_orders", true)

		activeA, err := store.Active(t.Context(), tenantA)
		require.NoError(t, err)
		require.Equal(t, []string{"variants"}, activeA)
		activeB, err := store.Active(t.Context(), tenantB)
		require.NoError(t, err)
		require.Equal(t, []string{"work_orders"}, activeB)
	})

	t.Run("RLS no deja escribir una fila de otro tenant", func(t *testing.T) {
		err := d.WithTenantTx(t.Context(), tenantB, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(),
				"INSERT INTO tenant_modules (tenant_id, module_id, enabled_at, enabled_by) VALUES ($1, 'lots_expiry', $2, $3)",
				tenantA, now, uuid.Must(uuid.NewV7()))
			return err
		})
		require.ErrorContains(t, err, "row-level security")
	})
}
