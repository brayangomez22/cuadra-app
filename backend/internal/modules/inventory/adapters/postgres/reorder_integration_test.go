//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func rule(t *testing.T, tenantID, variantID, locationID uuid.UUID, minQty, maxQty string) *domain.ReorderRule {
	t.Helper()
	r, err := domain.NewReorderRule(domain.ReorderRuleSpec{
		TenantID: tenantID, VariantID: variantID, LocationID: locationID, Min: dec(minQty), Max: dec(maxQty),
	})
	require.NoError(t, err)
	return r
}

func (f fixture) saveRule(t *testing.T, tenantID, variantID, locationID uuid.UUID, minQty, maxQty string) {
	t.Helper()
	require.NoError(t, f.reorder.Save(t.Context(), rule(t, tenantID, variantID, locationID, minQty, maxQty)))
}

func (f fixture) alerts(t *testing.T, tenantID uuid.UUID, locationID *uuid.UUID) domain.RuleStockPage {
	t.Helper()
	page, err := f.reorder.Alerts(t.Context(), tenantID, domain.ReorderFilter{LocationID: locationID, Limit: 100})
	require.NoError(t, err)
	return page
}

func TestReorderRules(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda la regla y la reemplaza", func(t *testing.T) {
		tenant := f.createTenant(t)
		variants := f.createProduct(t, tenant, 1)
		loc := f.createLocation(t, tenant, "Bodega")
		f.saveRule(t, tenant, variants[0], loc.ID(), "2", "6")
		f.saveRule(t, tenant, variants[0], loc.ID(), "3.5", "8")

		rules, err := f.reorder.ForVariants(t.Context(), tenant, loc.ID(), variants)
		require.NoError(t, err)
		require.Len(t, rules, 1)
		require.True(t, dec("3.5").Equal(rules[variants[0]].Min()))
		require.True(t, dec("8").Equal(rules[variants[0]].Max()))
	})

	t.Run("una variante o sede inexistente falla con su error", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.reorder.Save(t.Context(), rule(t, tenant, newID(), loc.ID(), "1", "2"))
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
		err = f.reorder.Save(t.Context(), rule(t, tenant, variant, newID(), "1", "2"))
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("elimina la regla; sin regla falla con ErrReorderRuleNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.saveRule(t, tenant, variant, loc.ID(), "1", "2")

		require.NoError(t, f.reorder.Delete(t.Context(), tenant, variant, loc.ID()))
		require.ErrorIs(t, f.reorder.Delete(t.Context(), tenant, variant, loc.ID()), domain.ErrReorderRuleNotFound)
	})
}

func TestReorderAlerts(t *testing.T) {
	f := newFixture(t)

	t.Run("la consulta coincide con la regla del dominio en los bordes", func(t *testing.T) {
		tenant := f.createTenant(t)
		loc := f.createLocation(t, tenant, "Bodega")
		cases := []struct {
			name, stock, min, max string
		}{
			{"sin fila de stock", "", "0", "5"},
			{"igual al mínimo", "10", "10", "50"},
			{"por debajo del mínimo, fraccionaria", "1.25", "2.5", "30"},
			{"por encima del mínimo", "10.0001", "10", "50"},
			{"sobre el máximo", "80", "10", "50"},
		}
		want := map[uuid.UUID]bool{}
		for _, tc := range cases {
			variant := f.createVariant(t, tenant)
			if tc.stock != "" {
				f.receive(t, tenant, variant, loc.ID(), tc.stock, "100")
			}
			f.saveRule(t, tenant, variant, loc.ID(), tc.min, tc.max)
			onHand := dec("0")
			if tc.stock != "" {
				onHand = dec(tc.stock)
			}
			want[variant] = rule(t, tenant, variant, loc.ID(), tc.min, tc.max).NeedsReorder(onHand)
		}

		page := f.alerts(t, tenant, new(loc.ID()))
		got := map[uuid.UUID]bool{}
		for _, item := range page.Items {
			got[item.Rule.VariantID()] = true
			require.True(t, item.Rule.NeedsReorder(item.OnHand))
		}
		for variant, needs := range want {
			require.Equal(t, needs, got[variant], variant)
		}
		require.Equal(t, 3, page.Total)
	})

	t.Run("pone primero lo agotado, luego lo más lejos de su mínimo en proporción, y filtra por sede", func(t *testing.T) {
		tenant := f.createTenant(t)
		north, south := f.createLocation(t, tenant, "Norte"), f.createLocation(t, tenant, "Sur")
		closer, further, empty := f.createVariant(t, tenant), f.createVariant(t, tenant), f.createVariant(t, tenant)
		// closer has 40 of 50 (80 %) and further 1 of 5 (20 %): further is
		// worse though closer lacks more units. empty, with none, goes first.
		f.receive(t, tenant, closer, north.ID(), "40", "100")
		f.receive(t, tenant, further, north.ID(), "1", "100")
		f.saveRule(t, tenant, closer, north.ID(), "50", "80")
		f.saveRule(t, tenant, further, north.ID(), "5", "10")
		f.saveRule(t, tenant, empty, north.ID(), "2", "4")
		f.saveRule(t, tenant, closer, south.ID(), "1", "3")

		page := f.alerts(t, tenant, new(north.ID()))
		require.Equal(t, 3, page.Total)
		require.Equal(t, empty, page.Items[0].Rule.VariantID())
		require.Equal(t, further, page.Items[1].Rule.VariantID())
		require.Equal(t, closer, page.Items[2].Rule.VariantID())
		require.Equal(t, 4, f.alerts(t, tenant, nil).Total)
	})

	t.Run("suma la existencia de todos los lotes", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, variant, loc.ID(), "3", "100")
		f.exec(t, tenant, `INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
			VALUES ($1, $2, $3, $4, 4, 100, $5)`, tenant, variant, loc.ID(), newID(), now)
		f.saveRule(t, tenant, variant, loc.ID(), "6", "12")

		require.Equal(t, 0, f.alerts(t, tenant, nil).Total)
		f.saveRule(t, tenant, variant, loc.ID(), "7", "12")
		page := f.alerts(t, tenant, nil)
		require.Equal(t, 1, page.Total)
		require.True(t, dec("7").Equal(page.Items[0].OnHand))
	})
}

func TestReorderTenantIsolation(t *testing.T) {
	f := newFixture(t)

	t.Run("aislamiento: el tenant B no ve ni modifica las reglas del tenant A", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		variant := f.createVariant(t, tenantA)
		loc := f.createLocation(t, tenantA, "Bodega")
		f.saveRule(t, tenantA, variant, loc.ID(), "1", "3")

		rules, err := f.reorder.ForVariants(t.Context(), tenantB, loc.ID(), []uuid.UUID{variant})
		require.NoError(t, err)
		require.Empty(t, rules)
		require.Equal(t, 0, f.alerts(t, tenantB, nil).Total)
		require.ErrorIs(t, f.reorder.Delete(t.Context(), tenantB, variant, loc.ID()), domain.ErrReorderRuleNotFound)
		err = f.reorder.Save(t.Context(), rule(t, tenantB, variant, loc.ID(), "0", "1"))
		require.ErrorIs(t, err, domain.ErrVariantNotFound)

		require.Equal(t, 1, f.alerts(t, tenantA, nil).Total)
	})
}
