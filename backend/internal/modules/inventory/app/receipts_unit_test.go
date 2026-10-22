package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// addUnit gives a variant of the fake catalog an alternate unit with factor
// and returns the unit's id.
func (e *env) addUnit(variantID uuid.UUID, factor string) uuid.UUID {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	id := newID()
	e.store.units[unitKey{variantID, id}] = dec(factor)
	return id
}

func TestReceiveStockInAlternateUnit(t *testing.T) {
	tests := []struct {
		name              string
		qty, cost, factor string
		wantQty, wantCost string
	}{
		{"ferretería: recibe 1 rollo de 100 m y suma 100 m al costo por metro", "1", "150000", "100", "100", "1500"},
		{"almacén de ropa: recibe 2 paquetes de 3 camisetas y suma 6 und", "2", "60000", "3", "6", "20000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			loc := e.createLocation(t, "Bodega")
			_, variants := e.addProduct(0)
			unit := e.addUnit(variants[0], tt.factor)

			m, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
				VariantID: variants[0], LocationID: loc.ID(), Quantity: dec(tt.qty), UnitCost: dec(tt.cost), UnitID: &unit,
			})
			require.NoError(t, err)

			requireDecimal(t, tt.wantQty, m.Quantity())
			requireDecimal(t, tt.wantCost, m.UnitCost())
			level := e.level(variants[0], loc.ID())
			requireDecimal(t, tt.wantQty, level.Quantity)
			requireDecimal(t, tt.wantCost, level.AverageCost)

			_, attrs := e.span(t, "inventory.ReceiveStock")
			require.True(t, attrs["unit.alternate"].AsBool())
		})
	}

	t.Run("sin unidad recibe en la unidad base", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)

		m := e.receive(t, variants[0], loc.ID(), "2.5", "1800")
		requireDecimal(t, "2.5", m.Quantity())
		_, attrs := e.span(t, "inventory.ReceiveStock")
		require.False(t, attrs["unit.alternate"].AsBool())
	})

	t.Run("rechaza una unidad que la variante no tiene y no mueve el stock", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(1)
		otherUnit := e.addUnit(variants[1], "12")

		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: variants[0], LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1000"), UnitID: &otherUnit,
		})
		require.ErrorIs(t, err, domain.ErrUnitNotAvailable)
		require.True(t, e.level(variants[0], loc.ID()).Quantity.IsZero())
	})

	t.Run("rechaza una cantidad que no cabe en la unidad base con 4 decimales", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)
		grams := e.addUnit(variants[0], "0.001")

		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: variants[0], LocationID: loc.ID(), Quantity: dec("0.25"), UnitCost: dec("30"), UnitID: &grams,
		})
		require.ErrorIs(t, err, domain.ErrQuantityPrecision)
	})
}
