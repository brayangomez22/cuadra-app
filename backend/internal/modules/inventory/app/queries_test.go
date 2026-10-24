package app_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestQueries(t *testing.T) {
	t.Run("lista las existencias de la sede", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()
		e.receive(t, variant, loc.ID(), "7", "500")

		page, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID()})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, variant, page.Items[0].VariantID())
		require.True(t, dec("7").Equal(page.Items[0].Quantity()))
	})

	t.Run("las existencias de una sede inexistente fallan con ErrLocationNotFound", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: newID()})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("el kardex devuelve los movimientos de la variante, el más reciente primero", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()
		first := e.receive(t, variant, loc.ID(), "1", "100")
		second := e.receive(t, variant, loc.ID(), "2", "100")
		e.receive(t, newID(), loc.ID(), "5", "100")

		page, err := e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{VariantID: variant})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Equal(t, second.ID(), page.Items[0].ID())
		require.Equal(t, first.ID(), page.Items[1].ID())
		_, attrs := e.span(t, "inventory.GetKardex")
		require.Equal(t, variant.String(), attrs["variant.id"].AsString())
	})

	t.Run("usa 20 resultados por defecto y limita la página a 100", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")

		for _, tc := range []struct{ limit, offset, wantLimit, wantOffset int }{
			{0, 0, 20, 0},
			{500, -3, 100, 0},
			{50, 10, 50, 10},
		} {
			_, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID(), Limit: tc.limit, Offset: tc.offset})
			require.NoError(t, err)
			_, err = e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{VariantID: newID(), Limit: tc.limit, Offset: tc.offset})
			require.NoError(t, err)

			gotStock := e.store.stockFilters[len(e.store.stockFilters)-1]
			gotKardex := e.store.kardexFilters[len(e.store.kardexFilters)-1]
			require.Equal(t, tc.wantLimit, gotStock.Limit)
			require.Equal(t, tc.wantOffset, gotStock.Offset)
			require.Equal(t, tc.wantLimit, gotKardex.Limit)
			require.Equal(t, tc.wantOffset, gotKardex.Offset)
		}
	})
}

func TestStockByVariant(t *testing.T) {
	t.Run("una entrada a una variante no cambia el stock de otra variante del mismo producto", func(t *testing.T) {
		// Ropa: la talla M y la talla L de la misma camiseta.
		e := newEnv(t)
		loc := e.createLocation(t, "Almacén")
		_, variants := e.addProduct(1)
		sizeM, sizeL := variants[0], variants[1]
		e.receive(t, sizeL, loc.ID(), "4", "15000")

		m := e.receive(t, sizeM, loc.ID(), "6", "15000")

		require.Equal(t, sizeM, m.VariantID())
		requireDecimal(t, "6", e.level(sizeM, loc.ID()).Quantity)
		requireDecimal(t, "4", e.level(sizeL, loc.ID()).Quantity)
	})

	t.Run("el span de un movimiento lleva la variante", func(t *testing.T) {
		// Ferretería: un producto simple, con solo su variante por defecto.
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)

		e.receive(t, variants[0], loc.ID(), "10", "28000")

		_, attrs := e.span(t, "inventory.ReceiveStock")
		require.Equal(t, variants[0].String(), attrs["variant.id"].AsString())
	})

	t.Run("el kardex de una variante reconstruye su stock", func(t *testing.T) {
		e := newEnv(t)
		from, to := e.createLocation(t, "Bodega"), e.createLocation(t, "Sede centro")
		_, variants := e.addProduct(1)
		v := variants[1]
		e.receive(t, v, from.ID(), "10", "1000")
		e.receive(t, variants[0], from.ID(), "99", "1000")
		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{VariantID: v, LocationID: from.ID(), Quantity: dec("-2.5"), Reason: "Conteo"})
		require.NoError(t, err)
		_, _, err = e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{VariantID: v, FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("3")})
		require.NoError(t, err)

		for _, loc := range []*domain.Location{from, to} {
			id := loc.ID()
			page, err := e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{VariantID: v, LocationID: &id})
			require.NoError(t, err)
			sum := decimal.Zero
			for _, m := range page.Items {
				require.Equal(t, v, m.VariantID())
				sum = sum.Add(m.Quantity())
			}
			requireDecimal(t, e.level(v, id).Quantity.String(), sum)
			requireDecimal(t, e.level(v, id).Quantity.String(), page.Items[0].BalanceAfter())
		}
		requireDecimal(t, "4.5", e.level(v, from.ID()).Quantity)
	})

	t.Run("dice el producto de cada variante, solo del tenant", func(t *testing.T) {
		e := newEnv(t)
		productID, variants := e.addProduct(1)
		unknown := newID()

		got, err := e.svc.ProductsOf(t.Context(), e.actor, append(slices.Clone(variants), unknown))
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]uuid.UUID{variants[0]: productID, variants[1]: productID}, got)

		got, err = e.svc.ProductsOf(t.Context(), app.Actor{TenantID: newID(), UserID: newID()}, variants)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
