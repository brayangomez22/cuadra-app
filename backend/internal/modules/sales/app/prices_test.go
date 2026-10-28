package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// testRule is a price rule of a module: it gives price to the variants in
// applies (to all if applies is nil), fails with err if set, and records
// the queries it sees.
type testRule struct {
	id      domain.PriceRuleID
	price   string
	applies map[uuid.UUID]bool
	err     error
	seen    *[]app.PriceQuery
}

func (r testRule) ID() domain.PriceRuleID { return r.id }

func (r testRule) Apply(_ context.Context, q app.PriceQuery) (decimal.Decimal, bool, error) {
	if r.seen != nil {
		*r.seen = append(*r.seen, q)
	}
	if r.err != nil {
		return decimal.Decimal{}, false, r.err
	}
	if r.applies != nil && !r.applies[q.VariantID] {
		return decimal.Decimal{}, false, nil
	}
	return dec(r.price), true, nil
}

func priceQuery(base string) app.PriceQuery {
	return app.PriceQuery{
		TenantID: newID(), LocationID: newID(), VariantID: newID(), ProductID: newID(),
		BasePrice: dec(base), Quantity: dec("1"),
	}
}

func TestPriceChain(t *testing.T) {
	t.Run("sin reglas aplica variant_price con el precio de la variante", func(t *testing.T) {
		chain, err := app.NewPriceChain()
		require.NoError(t, err)

		got, err := chain.Resolve(t.Context(), priceQuery("8403.36"))
		require.NoError(t, err)
		requireDec(t, "8403.36", got.UnitPrice)
		require.Equal(t, domain.VariantPriceRule, got.Rule)
	})

	t.Run("una regla registrada antes de la base da su precio sin redondear", func(t *testing.T) {
		chain, err := app.NewPriceChain(testRule{id: "price_lists", price: "9259.259259"})
		require.NoError(t, err)

		got, err := chain.Resolve(t.Context(), priceQuery("10000"))
		require.NoError(t, err)
		requireDec(t, "9259.259259", got.UnitPrice)
		require.Equal(t, domain.PriceRuleID("price_lists"), got.Rule)
	})

	t.Run("si ninguna regla aplica, cae en variant_price", func(t *testing.T) {
		chain, err := app.NewPriceChain(testRule{id: "price_lists", price: "1", applies: map[uuid.UUID]bool{}})
		require.NoError(t, err)

		got, err := chain.Resolve(t.Context(), priceQuery("10000"))
		require.NoError(t, err)
		requireDec(t, "10000", got.UnitPrice)
		require.Equal(t, domain.VariantPriceRule, got.Rule)
	})

	t.Run("las reglas se evalúan en el orden registrado y gana la primera que aplica", func(t *testing.T) {
		q := priceQuery("10000")
		var seenFirst, seenThird []app.PriceQuery
		chain, err := app.NewPriceChain(
			testRule{id: "promotions", price: "7000", applies: map[uuid.UUID]bool{}, seen: &seenFirst},
			testRule{id: "volume_pricing", price: "9000"},
			testRule{id: "price_lists", price: "8000", seen: &seenThird},
		)
		require.NoError(t, err)

		got, err := chain.Resolve(t.Context(), q)
		require.NoError(t, err)
		require.Equal(t, domain.PriceRuleID("volume_pricing"), got.Rule)
		requireDec(t, "9000", got.UnitPrice)
		require.Equal(t, []app.PriceQuery{q}, seenFirst, "la primera regla ve la consulta completa")
		require.Empty(t, seenThird, "después de la que aplica no se evalúa ninguna")
	})

	t.Run("propaga el error de una regla", func(t *testing.T) {
		boom := errors.New("boom")
		chain, err := app.NewPriceChain(testRule{id: "price_lists", err: boom})
		require.NoError(t, err)

		_, err = chain.Resolve(t.Context(), priceQuery("10000"))
		require.ErrorIs(t, err, boom)
	})

	t.Run("rechaza ids inválidos, duplicados o iguales a la base", func(t *testing.T) {
		tests := []struct {
			name  string
			rules []app.PriceRule
		}{
			{"id inválido", []app.PriceRule{testRule{id: "Lista Mayorista"}}},
			{"id duplicado", []app.PriceRule{testRule{id: "price_lists"}, testRule{id: "price_lists"}}},
			{"repite la base", []app.PriceRule{testRule{id: domain.VariantPriceRule}}},
			{"regla nula", []app.PriceRule{nil}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := app.NewPriceChain(tt.rules...)
				require.Error(t, err)
			})
		}
	})
}

func TestConfirmSalePrices(t *testing.T) {
	t.Run("la línea guarda el precio y el id de la regla que aplicó, y la base cubre el resto", func(t *testing.T) {
		var seen []app.PriceQuery
		rule := testRule{id: "price_lists", price: "9259.259259", applies: map[uuid.UUID]bool{}, seen: &seen}
		e := newEnv(t, rule)
		listed := e.addProduct("Camiseta básica", "10000", "0.19", true)
		rule.applies[listed] = true
		plain := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
		customer := e.addCustomer(true)
		location := newID()

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: location,
			CustomerID: &customer,
			Lines:      []app.LineRequest{lineReq(listed, "2", "0"), lineReq(plain, "1", "0")},
			Payments:   []app.PaymentRequest{cash("60000")},
		})
		require.NoError(t, err)

		lines := sale.Lines()
		require.Equal(t, domain.PriceRuleID("price_lists"), lines[0].PriceRule())
		requireDec(t, "9259.2593", lines[0].UnitPrice()) // la línea redondea a 4 decimales
		requireDec(t, "18518.52", lines[0].Gross())
		require.Equal(t, domain.VariantPriceRule, lines[1].PriceRule())
		requireDec(t, "30000", lines[1].UnitPrice())

		require.Len(t, seen, 2)
		require.Equal(t, app.PriceQuery{
			TenantID: e.actor.TenantID, LocationID: location, CustomerID: &customer,
			VariantID: listed, ProductID: e.productOf(listed), BasePrice: dec("10000"), Quantity: dec("2"),
			UnitFactor: domain.BaseUnit.Factor,
		}, seen[0])
	})

	t.Run("si una regla falla, la venta no se guarda ni reserva stock", func(t *testing.T) {
		boom := errors.New("boom")
		e := newEnv(t, testRule{id: "price_lists", err: boom})
		id := e.addProduct("Camiseta básica", "10000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{cash("11900")},
		})
		require.ErrorIs(t, err, boom)
		require.Zero(t, e.storedSales())
		require.Empty(t, e.store.reserved)
	})
}
