package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// sell confirms a sale of one product at location, paid in cash.
func (e *env) sell(t *testing.T, location uuid.UUID) *domain.Sale {
	t.Helper()
	p := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
	sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
		LocationID: location,
		Lines:      []app.LineRequest{lineReq(p, "1", "0")},
		Payments:   []app.PaymentRequest{cash("35700")},
	})
	require.NoError(t, err)
	return sale
}

func TestConfirmSaleNumbering(t *testing.T) {
	t.Run("registra el consecutivo en el span", func(t *testing.T) {
		e := newEnv(t)
		e.sell(t, newID())
		_, attrs := e.span(t, "sales.ConfirmSale")
		require.Equal(t, int64(1), attrs["sale.number"].AsInt64())
	})

	t.Run("asigna el consecutivo de cada sede", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		first := e.sell(t, location)
		second := e.sell(t, location)
		other := e.sell(t, newID())

		require.Equal(t, int64(1), first.Number())
		require.Equal(t, int64(2), second.Number())
		require.Equal(t, int64(1), other.Number())
	})

	t.Run("si falta stock, el error nombra el producto del catálogo", func(t *testing.T) {
		e := newEnv(t)
		p := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
		e.store.reserveErr = &domain.InsufficientStockError{ProductID: p}

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(p, "2", "0")},
			Payments:   []app.PaymentRequest{cash("71400")},
		})

		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var typed *domain.InsufficientStockError
		require.True(t, errors.As(err, &typed))
		require.Equal(t, p, typed.ProductID)
		require.Equal(t, "Cemento gris 50kg", typed.ProductName)
		require.Zero(t, e.storedSales())
	})
}

func TestVoidSale(t *testing.T) {
	t.Run("anula la venta, devuelve el stock y cuenta cuadra.sales.voided", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())

		voided, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "El cliente desistió")
		require.NoError(t, err)

		require.Equal(t, domain.SaleVoided, voided.Status())
		require.Equal(t, e.actor.UserID, voided.VoidInfo().UserID)
		require.Equal(t, "El cliente desistió", voided.VoidInfo().Reason)
		require.Equal(t, testNow.UTC(), voided.VoidInfo().At)
		require.Equal(t, []uuid.UUID{sale.ID()}, e.store.reversed)

		stored, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, stored.Status())

		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.voided"))
		span, attrs := e.span(t, "sales.VoidSale")
		require.Equal(t, codes.Unset, span.Status().Code)
		require.Equal(t, sale.ID().String(), attrs["sale.id"].AsString())
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("no devuelve el stock si la venta no está confirmada", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		_, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Primera")
		require.NoError(t, err)

		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Segunda")
		require.ErrorIs(t, err, domain.ErrSaleNotConfirmed)
		require.Len(t, e.store.reversed, 1)
		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.voided"))
	})

	t.Run("no devuelve el stock sin motivo", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		_, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), " ")
		require.ErrorIs(t, err, domain.ErrInvalidVoidReason)
		require.Empty(t, e.store.reversed)
	})

	t.Run("si falla la devolución de stock, la venta no queda anulada", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		boom := errors.New("boom")
		e.store.reverseErr = boom

		_, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "El cliente desistió")
		require.ErrorIs(t, err, boom)

		stored, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, stored.Status())
		require.Zero(t, e.counter(t, "cuadra.sales.voided"))
		span, _ := e.span(t, "sales.VoidSale")
		require.Equal(t, codes.Error, span.Status().Code)
	})

	t.Run("devuelve sale_not_found para una venta de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		other := e.actor
		other.TenantID = newID()

		_, err := e.svc.VoidSale(t.Context(), other, sale.ID(), "Ajena")
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
		require.Empty(t, e.store.reversed)
	})
}

func TestGetSale(t *testing.T) {
	t.Run("devuelve la venta con sus líneas y pagos", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())

		got, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Equal(t, sale.ID(), got.ID())
		require.Len(t, got.Lines(), 1)
		require.Len(t, got.Payments(), 1)
		_, attrs := e.span(t, "sales.GetSale")
		require.Equal(t, sale.ID().String(), attrs["sale.id"].AsString())
	})

	t.Run("devuelve sale_not_found para una venta de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		other := e.actor
		other.TenantID = newID()

		_, err := e.svc.GetSale(t.Context(), other, sale.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
	})
}

func TestListSales(t *testing.T) {
	bogota := time.FixedZone("COT", -5*3600)

	t.Run("lista las ventas del día en hora de Bogotá", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		e.sell(t, location)

		page, err := e.svc.ListSales(t.Context(), e.actor, app.ListSalesInput{
			Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), LocationID: &location, Limit: 20,
		})
		require.NoError(t, err)

		require.Equal(t, 1, page.Total)
		require.Len(t, e.store.filters, 1)
		f := e.store.filters[0]
		// El 10 de octubre en Bogotá va de las 05:00 UTC del 10 a las 05:00 UTC del 11.
		require.Equal(t, time.Date(2026, 10, 10, 5, 0, 0, 0, time.UTC), f.From.UTC())
		require.Equal(t, time.Date(2026, 10, 11, 5, 0, 0, 0, time.UTC), f.To.UTC())
		require.Equal(t, &location, f.LocationID)
		require.Equal(t, 20, f.Limit)
		_, attrs := e.span(t, "sales.ListSales")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("sin fecha lista las de hoy en Bogotá", func(t *testing.T) {
		e := newEnv(t)
		// 23:30 del 9 en Bogotá es 04:30 UTC del 10: el día sigue siendo el 9.
		e.svc.Now = func() time.Time { return time.Date(2026, 10, 10, 4, 30, 0, 0, time.UTC) }

		_, err := e.svc.ListSales(t.Context(), e.actor, app.ListSalesInput{Limit: 20})
		require.NoError(t, err)

		f := e.store.filters[0]
		require.Equal(t, time.Date(2026, 10, 9, 0, 0, 0, 0, bogota).UTC(), f.From.UTC())
		require.Equal(t, 24*time.Hour, f.To.Sub(f.From))
	})

	t.Run("no lista ventas de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		e.sell(t, newID())
		other := e.actor
		other.TenantID = newID()

		page, err := e.svc.ListSales(t.Context(), other, app.ListSalesInput{Date: testNow, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, page.Total)
	})
}
