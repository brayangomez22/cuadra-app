//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/postgres"
)

var now = time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)

// createTenant stores a tenant row directly: tenant_feature_flags references it.
func createTenant(t *testing.T, d *db.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, d.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, $2, '900123456-8', 'active', 'generica', $3)",
			id, name, now)
		return err
	}))
	return id
}

func change(key string, on bool) flags.Change {
	return flags.Change{Key: key, Enabled: on, By: uuid.Must(uuid.NewV7()), At: now}
}

func TestStoreGlobals(t *testing.T) {
	d := dbtest.New(t)
	store := postgres.NewStore(d)

	t.Run("guarda y cambia el valor global de una flag", func(t *testing.T) {
		require.NoError(t, store.SetGlobal(t.Context(), change("global_on_off", true)))
		got, err := store.Globals(t.Context())
		require.NoError(t, err)
		require.True(t, got["global_on_off"])

		require.NoError(t, store.SetGlobal(t.Context(), change("global_on_off", false)))
		got, err = store.Globals(t.Context())
		require.NoError(t, err)
		require.False(t, got["global_on_off"])
		require.Contains(t, got, "global_on_off")
	})
}

func TestStoreExceptions(t *testing.T) {
	d := dbtest.New(t)
	store := postgres.NewStore(d)

	t.Run("guarda, cambia y quita la excepción de un tenant", func(t *testing.T) {
		tenant := createTenant(t, d, "Ferretería La Tuerca")

		require.NoError(t, store.SetException(t.Context(), tenant, change("new_pos", true)))
		require.NoError(t, store.SetException(t.Context(), tenant, change("quick_sale", false)))
		got, err := store.Exceptions(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, map[string]bool{"new_pos": true, "quick_sale": false}, got)

		require.NoError(t, store.ClearException(t.Context(), tenant, "new_pos"))
		require.NoError(t, store.ClearException(t.Context(), tenant, "new_pos"), "quitarla otra vez no falla")
		got, err = store.Exceptions(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, map[string]bool{"quick_sale": false}, got)
	})

	t.Run("rechaza la excepción de un tenant que no existe", func(t *testing.T) {
		err := store.SetException(t.Context(), uuid.Must(uuid.NewV7()), change("new_pos", true))
		require.ErrorIs(t, err, flags.ErrUnknownTenant)
	})

	t.Run("lista las excepciones de todos los tenants con su nombre", func(t *testing.T) {
		hardware := createTenant(t, d, "Ferretería El Tornillo")
		clothing := createTenant(t, d, "Calzado Andino")
		require.NoError(t, store.SetException(t.Context(), hardware, change("listed", true)))
		require.NoError(t, store.SetException(t.Context(), clothing, change("listed", false)))

		all, err := store.AllExceptions(t.Context())
		require.NoError(t, err)
		require.Contains(t, all, flags.Exception{TenantID: hardware, TenantName: "Ferretería El Tornillo", Key: "listed", Enabled: true})
		require.Contains(t, all, flags.Exception{TenantID: clothing, TenantName: "Calzado Andino", Key: "listed", Enabled: false})
	})
}

func TestStoreIsolation(t *testing.T) {
	d := dbtest.New(t)
	store := postgres.NewStore(d)

	t.Run("aislamiento: el tenant B no ve ni modifica las excepciones del A", func(t *testing.T) {
		a := createTenant(t, d, "Ferretería La Tuerca")
		b := createTenant(t, d, "Calzado Andino")
		require.NoError(t, store.SetException(t.Context(), a, change("new_pos", true)))

		got, err := store.Exceptions(t.Context(), b)
		require.NoError(t, err)
		require.Empty(t, got)

		// B's transaction cannot touch A's row, even naming A's tenant_id.
		var cleared int64
		require.NoError(t, d.WithTenantTx(t.Context(), b, func(tx pgx.Tx) error {
			tag, err := tx.Exec(t.Context(), "DELETE FROM tenant_feature_flags WHERE tenant_id = $1", a)
			cleared = tag.RowsAffected()
			return err
		}))
		require.Zero(t, cleared)
		err = d.WithTenantTx(t.Context(), b, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(),
				"INSERT INTO tenant_feature_flags (tenant_id, key, enabled, updated_at, updated_by) VALUES ($1, 'new_pos', false, now(), $2)",
				a, uuid.Must(uuid.NewV7()))
			return err
		})
		require.Error(t, err, "RLS rechaza escribir una fila de otro tenant")

		got, err = store.Exceptions(t.Context(), a)
		require.NoError(t, err)
		require.Equal(t, map[string]bool{"new_pos": true}, got)
	})

	t.Run("la app no lee las demás columnas de los tenants por la función de las flags", func(t *testing.T) {
		err := d.WithTx(t.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "SET LOCAL ROLE flag_lookup")
			return err
		})
		require.Error(t, err, "app_user no puede asumir flag_lookup")
	})
}
