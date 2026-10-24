package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

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
