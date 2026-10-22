package app_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// addUnit gives a variant of the actor's catalog an alternate unit and
// returns the unit's id.
func (e *env) addUnit(variantID uuid.UUID, code, factor, price string) uuid.UUID {
	id := newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	key := catalogKey{e.actor.TenantID, variantID}
	v := e.store.variants[key]
	v.Units = append(v.Units, app.UnitInfo{UnitID: id, Code: code, Factor: dec(factor), Price: dec(price)})
	e.store.variants[key] = v
	return id
}

func unitLine(variantID, unitID uuid.UUID, qty string) app.LineRequest {
	l := lineReq(variantID, qty, "0")
	l.UnitID = &unitID
	return l
}

// queryRecorder is a price rule that never applies and records what it
// was asked.
type queryRecorder struct{ queries *[]app.PriceQuery }

func (queryRecorder) ID() domain.PriceRuleID { return "recorder" }

func (r queryRecorder) Apply(_ context.Context, q app.PriceQuery) (decimal.Decimal, bool, error) {
	*r.queries = append(*r.queries, q)
	return decimal.Decimal{}, false, nil
}

func TestConfirmSaleInAlternateUnit(t *testing.T) {
	tests := []struct {
		name                string
		product, basePrice  string
		unit, factor, price string
		baseQty, unitQty    string
		wantReserved        string
		wantTotal           string
	}{
		{
			name: "ferretería: 2,5 m sueltos y 1 rollo de 100 m del mismo cable", product: "Cable #12", basePrice: "1800",
			unit: "rollo", factor: "100", price: "150000", baseQty: "2.5", unitQty: "1",
			// 4500 + 150000 = 154500 base; IVA 29355
			wantReserved: "102.5", wantTotal: "183855",
		},
		{
			name: "almacén de ropa: 1 camiseta suelta y 2 paquetes de 3", product: "Camiseta básica", basePrice: "30000",
			unit: "paquete", factor: "3", price: "75000", baseQty: "1", unitQty: "2",
			// 30000 + 150000 = 180000 base; IVA 34200
			wantReserved: "7", wantTotal: "214200",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("vende al precio de la unidad y reserva la cantidad en unidad base, sumada por variante", func(t *testing.T) {
				e := newEnv(t)
				variant := e.addProduct(tt.product, tt.basePrice, "0.19", true)
				unit := e.addUnit(variant, tt.unit, tt.factor, tt.price)

				sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
					LocationID: newID(),
					Lines:      []app.LineRequest{lineReq(variant, tt.baseQty, "0"), unitLine(variant, unit, tt.unitQty)},
					Payments:   []app.PaymentRequest{cash(tt.wantTotal)},
				})
				require.NoError(t, err)

				lines := sale.Lines()
				require.Equal(t, domain.BaseUnit, lines[0].Unit())
				requireDec(t, tt.basePrice, lines[0].UnitPrice())
				require.Equal(t, tt.unit, lines[1].Unit().Code)
				requireDec(t, tt.factor, lines[1].Unit().Factor)
				requireDec(t, tt.unitQty, lines[1].Quantity())
				requireDec(t, tt.price, lines[1].UnitPrice())
				require.Equal(t, domain.VariantPriceRule, lines[1].PriceRule())
				requireDec(t, tt.wantTotal, sale.Totals().Total)

				require.Len(t, e.store.reserved, 1)
				reserved := e.store.reserved[0].lines
				require.Len(t, reserved, 1, "una línea de stock por variante")
				requireDec(t, tt.wantReserved, reserved[0].Quantity)

				require.Equal(t, int64(1), e.counter(t, "cuadra.sales.alternate_unit_lines"))
			})
		})
	}

	t.Run("pasa la unidad y su precio a las reglas de precio", func(t *testing.T) {
		var queries []app.PriceQuery
		e := newEnv(t, queryRecorder{&queries})
		variant := e.addProduct("Tornillo 1/4", "500", "0.19", true)
		unit := e.addUnit(variant, "caja", "12", "5500")

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(variant, "3", "0"), unitLine(variant, unit, "1")},
			Payments:   []app.PaymentRequest{cash("10000")},
		})
		require.NoError(t, err)

		require.Len(t, queries, 2)
		require.Nil(t, queries[0].UnitID)
		requireDec(t, "1", queries[0].UnitFactor)
		requireDec(t, "500", queries[0].BasePrice)
		require.Equal(t, &unit, queries[1].UnitID)
		requireDec(t, "12", queries[1].UnitFactor)
		requireDec(t, "5500", queries[1].BasePrice)
		requireDec(t, "1", queries[1].Quantity)
	})

	t.Run("rechaza una unidad que la variante no tiene y no guarda nada", func(t *testing.T) {
		e := newEnv(t)
		variant := e.addProduct("Tornillo 1/4", "500", "0.19", true)
		other := e.addProduct("Camiseta básica", "30000", "0.19", true)
		otherUnit := e.addUnit(other, "paquete", "3", "75000")

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{unitLine(variant, otherUnit, "1")},
			Payments:   []app.PaymentRequest{cash("100000")},
		})
		require.ErrorIs(t, err, domain.ErrUnitNotAvailable)
		require.Zero(t, e.storedSales())
		require.Empty(t, e.store.reserved)
		require.Zero(t, e.counter(t, "cuadra.sales.alternate_unit_lines"))
	})

	t.Run("rechaza una cantidad que no cabe en la unidad base con 4 decimales", func(t *testing.T) {
		e := newEnv(t)
		variant := e.addProduct("Café molido", "40000", "0.05", true)
		grams := e.addUnit(variant, "g", "0.001", "40")

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{unitLine(variant, grams, "0.25")},
			Payments:   []app.PaymentRequest{cash("100")},
		})
		require.ErrorIs(t, err, domain.ErrQuantityPrecision)
		require.Zero(t, e.storedSales())
	})
}
