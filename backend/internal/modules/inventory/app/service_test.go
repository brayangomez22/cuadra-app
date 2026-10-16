package app_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func (e *env) createLocation(t *testing.T, name string) *domain.Location {
	t.Helper()
	l, err := e.svc.CreateLocation(t.Context(), e.actor, name)
	require.NoError(t, err)
	return l
}

func (e *env) receive(t *testing.T, variantID, locationID uuid.UUID, qty, cost string) *domain.StockMovement {
	t.Helper()
	m, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
		VariantID: variantID, LocationID: locationID, Quantity: dec(qty), UnitCost: dec(cost),
	})
	require.NoError(t, err)
	return m
}

// deactivate takes a stored location out of use; there is no use case for it yet.
func (e *env) deactivate(id uuid.UUID) {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.locations[id].Deactivate()
}

func TestNewService(t *testing.T) {
	t.Run("falla si falta un repositorio", func(t *testing.T) {
		_, err := app.NewService(app.Deps{})
		require.Error(t, err)
	})
}

func TestLocations(t *testing.T) {
	t.Run("crea una sede activa", func(t *testing.T) {
		e := newEnv(t)

		l, err := e.svc.CreateLocation(t.Context(), e.actor, " Bodega principal ")
		require.NoError(t, err)

		require.Equal(t, e.actor.TenantID, l.TenantID())
		require.Equal(t, "Bodega principal", l.Name())
		require.True(t, l.IsActive())
		require.Equal(t, testNow, l.CreatedAt())
		_, attrs := e.span(t, "inventory.CreateLocation")
		require.Equal(t, l.ID().String(), attrs["location.id"].AsString())
	})

	t.Run("rechaza sede con nombre repetido", func(t *testing.T) {
		e := newEnv(t)
		e.createLocation(t, "Bodega principal")

		_, err := e.svc.CreateLocation(t.Context(), e.actor, "bodega PRINCIPAL")
		require.ErrorIs(t, err, domain.ErrLocationNameTaken)
	})

	t.Run("rechaza nombre vacío", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.CreateLocation(t.Context(), e.actor, "  ")
		require.ErrorIs(t, err, domain.ErrInvalidLocationName)
	})

	t.Run("lista las sedes del tenant por nombre", func(t *testing.T) {
		e := newEnv(t)
		e.createLocation(t, "Sede norte")
		e.createLocation(t, "Bodega")

		list, err := e.svc.ListLocations(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Bodega", list[0].Name())
		require.Equal(t, "Sede norte", list[1].Name())
	})
}

func TestReceiveStock(t *testing.T) {
	t.Run("registra una entrada y actualiza el nivel con el costo promedio", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()

		e.receive(t, variant, loc.ID(), "10", "1000")
		m := e.receive(t, variant, loc.ID(), "10", "1200")

		require.Equal(t, domain.MovementPurchaseIn, m.Type())
		require.True(t, dec("10").Equal(m.Quantity()))
		require.True(t, dec("1200").Equal(m.UnitCost()))
		require.True(t, dec("20").Equal(m.BalanceAfter()))
		require.True(t, dec("1100").Equal(m.AverageCostAfter()))
		require.Equal(t, e.actor.UserID, m.UserID())
		require.Equal(t, e.actor.TenantID, m.TenantID())
		require.Equal(t, testNow, m.OccurredAt())
		level := e.level(variant, loc.ID())
		require.True(t, dec("20").Equal(level.Quantity))
		require.True(t, dec("1100").Equal(level.AverageCost))
	})

	t.Run("rechaza entrada en una sede inexistente", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: newID(), LocationID: newID(), Quantity: dec("1"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("rechaza entrada en una sede de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		other := app.Actor{TenantID: newID(), UserID: newID()}

		_, err := e.svc.ReceiveStock(t.Context(), other, app.ReceiptInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("rechaza entrada en una sede inactiva", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		e.deactivate(loc.ID())

		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
	})

	t.Run("rechaza cantidad cero", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("0"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("cuenta el movimiento con su tipo y abre su span con variante y sede", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()

		e.receive(t, variant, loc.ID(), "2.5", "3000")

		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.movements", "type", "purchase_in"))
		_, attrs := e.span(t, "inventory.ReceiveStock")
		require.Equal(t, variant.String(), attrs["variant.id"].AsString())
		require.Equal(t, loc.ID().String(), attrs["location.id"].AsString())
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, e.actor.UserID.String(), attrs["user.id"].AsString())
	})
}

func TestAdjustStock(t *testing.T) {
	t.Run("un ajuste positivo suma al stock sin cambiar el costo promedio", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()
		e.receive(t, variant, loc.ID(), "10", "1000")

		m, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variant, LocationID: loc.ID(), Quantity: dec("2"), Reason: "Conteo físico",
		})
		require.NoError(t, err)

		require.Equal(t, domain.MovementAdjustment, m.Type())
		require.Equal(t, "Conteo físico", m.Reason())
		level := e.level(variant, loc.ID())
		require.True(t, dec("12").Equal(level.Quantity))
		require.True(t, dec("1000").Equal(level.AverageCost))
		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.movements", "type", "adjustment"))
	})

	t.Run("un ajuste negativo mayor al stock falla con ErrInsufficientStock y cuenta en la métrica", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		variant := newID()
		e.receive(t, variant, loc.ID(), "5", "1000")

		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variant, LocationID: loc.ID(), Quantity: dec("-6"), Reason: "Bulto roto",
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		require.True(t, dec("5").Equal(e.level(variant, loc.ID()).Quantity))
		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.insufficient_stock", "type", "adjustment"))
		require.Zero(t, e.counter(t, "cuadra.inventory.movements", "type", "adjustment"))
		span, _ := e.span(t, "inventory.AdjustStock")
		require.Equal(t, codes.Error, span.Status().Code)
	})

	t.Run("un ajuste negativo pasa si el tenant permite stock negativo", func(t *testing.T) {
		e := newEnv(t)
		e.store.policies[e.actor.TenantID] = domain.StockPolicy{AllowNegative: true}
		loc := e.createLocation(t, "Bodega")
		variant := newID()

		m, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variant, LocationID: loc.ID(), Quantity: dec("-3"), Reason: "Venta sin registrar",
		})
		require.NoError(t, err)
		require.True(t, dec("-3").Equal(m.BalanceAfter()))
	})

	t.Run("un ajuste sin motivo se rechaza", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")

		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("1"), Reason: " ",
		})
		require.ErrorIs(t, err, domain.ErrAdjustmentReasonRequired)
	})

	t.Run("rechaza ajuste en una sede inactiva", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		e.deactivate(loc.ID())

		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("1"), Reason: "Conteo",
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
	})
}

func TestTransferStock(t *testing.T) {
	t.Run("mueve el stock entre sedes y genera dos movimientos que suman cero", func(t *testing.T) {
		e := newEnv(t)
		from := e.createLocation(t, "Bodega")
		to := e.createLocation(t, "Sede centro")
		variant := newID()
		e.receive(t, variant, from.ID(), "10", "1000")

		out, in, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			VariantID: variant, FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("4"),
		})
		require.NoError(t, err)

		require.Equal(t, domain.MovementTransferOut, out.Type())
		require.Equal(t, domain.MovementTransferIn, in.Type())
		require.True(t, out.Quantity().Add(in.Quantity()).IsZero())
		require.Equal(t, out.Reference(), in.Reference())
		require.True(t, dec("6").Equal(e.level(variant, from.ID()).Quantity))
		toLevel := e.level(variant, to.ID())
		require.True(t, dec("4").Equal(toLevel.Quantity))
		require.True(t, dec("1000").Equal(toLevel.AverageCost))
		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.movements", "type", "transfer_out"))
		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.movements", "type", "transfer_in"))
	})

	t.Run("un traslado a la misma sede se rechaza", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")

		_, _, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			VariantID: newID(), FromLocationID: loc.ID(), ToLocationID: loc.ID(), Quantity: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrSameLocationTransfer)
	})

	t.Run("un traslado mayor al stock falla y no cambia ninguna sede", func(t *testing.T) {
		e := newEnv(t)
		from := e.createLocation(t, "Bodega")
		to := e.createLocation(t, "Sede centro")
		variant := newID()
		e.receive(t, variant, from.ID(), "3", "1000")

		_, _, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			VariantID: variant, FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("4"),
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		require.True(t, dec("3").Equal(e.level(variant, from.ID()).Quantity))
		require.True(t, e.level(variant, to.ID()).Quantity.IsZero())
		require.EqualValues(t, 1, e.counter(t, "cuadra.inventory.insufficient_stock", "type", "transfer_out"))
	})

	t.Run("rechaza traslado hacia una sede inactiva", func(t *testing.T) {
		e := newEnv(t)
		from := e.createLocation(t, "Bodega")
		to := e.createLocation(t, "Sede centro")
		e.deactivate(to.ID())
		variant := newID()
		e.receive(t, variant, from.ID(), "3", "1000")

		_, _, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			VariantID: variant, FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
	})
}

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

	t.Run("una entrada con product_id va a la variante por defecto del producto", func(t *testing.T) {
		// Ferretería: un producto simple, con solo su variante por defecto.
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		productID, variants := e.addProduct(0)

		m, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			ProductID: productID, LocationID: loc.ID(), Quantity: dec("10"), UnitCost: dec("28000"),
		})

		require.NoError(t, err)
		require.Equal(t, variants[0], m.VariantID())
		requireDecimal(t, "10", e.level(variants[0], loc.ID()).Quantity)
		_, attrs := e.span(t, "inventory.ReceiveStock")
		require.Equal(t, variants[0].String(), attrs["variant.id"].AsString())
	})

	t.Run("un ajuste y un traslado con product_id también van a la variante por defecto", func(t *testing.T) {
		e := newEnv(t)
		from, to := e.createLocation(t, "Bodega"), e.createLocation(t, "Sede centro")
		productID, variants := e.addProduct(1)
		e.receive(t, variants[0], from.ID(), "10", "1000")

		adj, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			ProductID: productID, LocationID: from.ID(), Quantity: dec("-1"), Reason: "Conteo",
		})
		require.NoError(t, err)
		out, in, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			ProductID: productID, FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("4"),
		})
		require.NoError(t, err)

		for _, m := range []*domain.StockMovement{adj, out, in} {
			require.Equal(t, variants[0], m.VariantID())
		}
		requireDecimal(t, "5", e.level(variants[0], from.ID()).Quantity)
		requireDecimal(t, "4", e.level(variants[0], to.ID()).Quantity)
	})

	t.Run("una entrada con product_id de un producto inexistente falla con ErrProductNotFound", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			ProductID: newID(), LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("no resuelve el producto de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		productID, _ := e.addProduct(0)
		other := app.Actor{TenantID: newID(), UserID: newID()}
		loc, err := e.svc.CreateLocation(t.Context(), other, "Bodega")
		require.NoError(t, err)
		_, err = e.svc.ReceiveStock(t.Context(), other, app.ReceiptInput{
			ProductID: productID, LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("rechaza un movimiento sin variante ni producto, o con ambos", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		productID, variants := e.addProduct(0)
		for name, in := range map[string]app.ReceiptInput{
			"sin variante ni producto": {LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1")},
			"con ambos":                {VariantID: variants[0], ProductID: productID, LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1")},
		} {
			_, err := e.svc.ReceiveStock(t.Context(), e.actor, in)
			require.ErrorIs(t, err, domain.ErrVariantRequired, name)
		}
		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{LocationID: loc.ID(), Quantity: dec("1"), Reason: "Conteo"})
		require.ErrorIs(t, err, domain.ErrVariantRequired)
		_, _, err = e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			FromLocationID: loc.ID(), ToLocationID: newID(), Quantity: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrVariantRequired)
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

	t.Run("el kardex por producto devuelve el de su variante por defecto; si el producto no existe, está vacío", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		productID, variants := e.addProduct(1)
		e.receive(t, variants[0], loc.ID(), "1", "100")
		e.receive(t, variants[1], loc.ID(), "2", "100")

		page, err := e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{ProductID: productID})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, variants[0], page.Items[0].VariantID())

		page, err = e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{ProductID: newID()})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		require.Empty(t, page.Items)
	})

	t.Run("las existencias filtradas por producto son las de su variante por defecto", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		productID, variants := e.addProduct(1)
		e.receive(t, variants[0], loc.ID(), "1", "100")
		e.receive(t, variants[1], loc.ID(), "2", "100")

		page, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID(), ProductID: &productID})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, variants[0], page.Items[0].VariantID())

		missing := newID()
		page, err = e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID(), ProductID: &missing})
		require.NoError(t, err)
		require.Zero(t, page.Total)
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
