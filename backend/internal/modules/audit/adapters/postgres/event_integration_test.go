//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

func TestEventRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee el evento con sus details", func(t *testing.T) {
		tenant, user := f.createTenant(t, false), newID()
		stored := priceChanged(t, tenant, user, now, "4500.00", "4800.00")
		require.NoError(t, f.events.Append(t.Context(), stored))
		require.NoError(t, f.events.Append(t.Context(), moduleEnabled(t, tenant, user, now.Add(time.Minute))))

		page := f.list(t, domain.EventFilter{TenantID: tenant})
		require.Equal(t, 2, page.Total)
		require.Equal(t, domain.ActionModuleEnabled, page.Items[0].Action())
		require.Nil(t, page.Items[0].EntityID())
		got := page.Items[1]
		require.Equal(t, stored.Snapshot(), got.Snapshot())
		require.Equal(t, "variant", got.EntityType())
	})

	t.Run("filtra por acción, usuario, entidad y rango, y pagina", func(t *testing.T) {
		tenant, user, other := f.createTenant(t, false), newID(), newID()
		first := priceChanged(t, tenant, user, now, "1.00", "2.00")
		for _, e := range []*domain.Event{
			first,
			priceChanged(t, tenant, other, now.Add(time.Hour), "2.00", "3.00"),
			moduleEnabled(t, tenant, user, now.Add(2*time.Hour)),
		} {
			require.NoError(t, f.events.Append(t.Context(), e))
		}
		action := domain.ActionVariantPriceChanged
		require.Equal(t, 2, f.list(t, domain.EventFilter{TenantID: tenant, Action: &action}).Total)
		require.Equal(t, 2, f.list(t, domain.EventFilter{TenantID: tenant, UserID: &user}).Total)
		require.Equal(t, 1, f.list(t, domain.EventFilter{TenantID: tenant, EntityID: first.EntityID()}).Total)
		from, before := now.Add(30*time.Minute), now.Add(90*time.Minute)
		require.Equal(t, 1, f.list(t, domain.EventFilter{TenantID: tenant, OccurredFrom: &from, OccurredBefore: &before}).Total)

		page := f.list(t, domain.EventFilter{TenantID: tenant, Limit: 1, Offset: 2})
		require.Equal(t, 3, page.Total)
		require.Equal(t, first.ID(), page.Items[0].ID())
	})

	t.Run("si la operación se revierte, no queda evento", func(t *testing.T) {
		tenant := f.createTenant(t, false)
		failed := errors.New("la operación falló")
		err := f.db.InTenantTx(t.Context(), tenant, func(ctx context.Context, _ pgx.Tx) error {
			require.NoError(t, f.events.Append(ctx, priceChanged(t, tenant, newID(), now, "1.00", "2.00")))
			return failed
		})
		require.ErrorIs(t, err, failed)
		require.Zero(t, f.list(t, domain.EventFilter{TenantID: tenant}).Total)
	})

	t.Run("aislamiento: el tenant B no ve los eventos del tenant A", func(t *testing.T) {
		a, b := f.createTenant(t, false), f.createTenant(t, false)
		require.NoError(t, f.events.Append(t.Context(), priceChanged(t, a, newID(), now, "1.00", "2.00")))
		require.Zero(t, f.list(t, domain.EventFilter{TenantID: b}).Total)
		// Nor can B write an event of A's.
		err := f.db.WithTenantTx(t.Context(), b, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(),
				`INSERT INTO audit_events (id, tenant_id, user_id, role, action, entity_type, occurred_at, details)
				 VALUES ($1, $2, $3, 'owner', 'module.enabled', 'module', $4, '{"module":"variants"}')`,
				newID(), a, newID(), now)
			return err
		})
		require.ErrorContains(t, err, "row-level security")
	})

	t.Run("el rol de la app no puede modificar ni borrar eventos", func(t *testing.T) {
		tenant := f.createTenant(t, false)
		require.NoError(t, f.events.Append(t.Context(), priceChanged(t, tenant, newID(), now, "1.00", "2.00")))
		for _, stmt := range []string{
			`UPDATE audit_events SET details = '{"old_price":"9.00","new_price":"2.00"}'`,
			`DELETE FROM audit_events`,
		} {
			err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
				_, err := tx.Exec(t.Context(), stmt)
				return err
			})
			require.ErrorContains(t, err, "permission denied", stmt)
		}
		require.Equal(t, 1, f.list(t, domain.EventFilter{TenantID: tenant}).Total)
	})

	t.Run("la purga de la demo borra también sus eventos", func(t *testing.T) {
		tenant := f.createTenant(t, true)
		require.NoError(t, f.events.Append(t.Context(), moduleEnabled(t, tenant, newID(), now)))
		require.NoError(t, f.db.WithTx(t.Context(), func(tx pgx.Tx) error {
			var result string
			return tx.QueryRow(t.Context(), "SELECT purge_demo_tenant($1)", tenant).Scan(&result)
		}))
		require.Zero(t, f.list(t, domain.EventFilter{TenantID: tenant}).Total)
	})
}
