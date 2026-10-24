//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleRepositoryList(t *testing.T) {
	f := newFixture(t)

	t.Run("lista las ventas del rango de la más reciente a la más antigua, con sede y paginación", func(t *testing.T) {
		tn := f.createTenant(t)
		other := f.createLocation(t, tn.id)
		day := time.Date(2026, 10, 10, 5, 0, 0, 0, time.UTC)
		early := confirmed(t, tn, tn.location, day.Add(time.Hour))
		late := confirmed(t, tn, tn.location, day.Add(20*time.Hour))
		elsewhere := confirmed(t, tn, other, day.Add(2*time.Hour))
		before := confirmed(t, tn, tn.location, day.Add(-time.Second))
		next := confirmed(t, tn, tn.location, day.Add(24*time.Hour))
		for _, s := range []*domain.Sale{early, late, elsewhere, before, next} {
			f.save(t, s)
		}
		_, err := f.sales.Void(t.Context(), tn.id, early.ID(), func(_ context.Context, s *domain.Sale) error {
			return s.Void(newID(), "Error", day.Add(3*time.Hour))
		})
		require.NoError(t, err)

		page, err := f.sales.List(t.Context(), tn.id, domain.SaleFilter{From: day, To: day.Add(24 * time.Hour), Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 3, page.Total)
		ids := []uuid.UUID{page.Items[0].ID, page.Items[1].ID, page.Items[2].ID}
		require.Equal(t, []uuid.UUID{late.ID(), elsewhere.ID(), early.ID()}, ids)

		voided := page.Items[2]
		require.Equal(t, domain.SaleVoided, voided.Status)
		require.NotNil(t, voided.VoidedAt)
		require.True(t, dec("123760").Equal(voided.Total), "total %s", voided.Total)
		require.Equal(t, early.Number(), voided.Number)
		require.Equal(t, early.UserID(), voided.UserID)
		require.Equal(t, tn.location, voided.LocationID)
		require.Equal(t, day.Add(time.Hour), voided.ConfirmedAt)

		page, err = f.sales.List(t.Context(), tn.id, domain.SaleFilter{From: day, To: day.Add(24 * time.Hour), LocationID: &tn.location, Limit: 1, Offset: 1})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Len(t, page.Items, 1)
		require.Equal(t, early.ID(), page.Items[0].ID)
	})

	t.Run("rechaza un límite de página fuera de rango", func(t *testing.T) {
		tn := f.createTenant(t)
		for _, filter := range []domain.SaleFilter{{Limit: 0}, {Limit: 101}, {Limit: 10, Offset: -1}} {
			filter.From, filter.To = now, now.Add(time.Hour)
			_, err := f.sales.List(t.Context(), tn.id, filter)
			require.Error(t, err)
		}
	})
}

func TestSaleRepositoryIsolation(t *testing.T) {
	f := newFixture(t)

	t.Run("aislamiento: el tenant B no ve, no lista ni anula ventas del tenant A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := confirmed(t, a, a.location, now)
		f.save(t, s)

		_, err := f.sales.Get(t.Context(), b.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)

		page, err := f.sales.List(t.Context(), b.id, domain.SaleFilter{From: now.Add(-time.Hour), To: now.Add(time.Hour), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, page.Total)

		_, err = f.sales.Void(t.Context(), b.id, s.ID(), func(_ context.Context, s *domain.Sale) error {
			return s.Void(newID(), "Ajena", now)
		})
		require.ErrorIs(t, err, domain.ErrSaleNotFound)

		got, err := f.sales.Get(t.Context(), a.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, got.Status())
	})

	t.Run("aislamiento: el consecutivo del tenant B empieza en 1 aunque A ya vendió", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.save(t, confirmed(t, a, a.location, now))
		f.save(t, confirmed(t, a, a.location, now))
		s := confirmed(t, b, b.location, now)
		f.save(t, s)
		require.Equal(t, int64(1), s.Number())
	})
}

func TestPolicyRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("sin configuración el tope de descuento es 0", func(t *testing.T) {
		tn := f.createTenant(t)
		p, err := f.policies.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.False(t, p.MaxDiscount.Unlimited())
		require.True(t, p.MaxDiscount.Max().IsZero())
	})

	t.Run("devuelve el tope configurado solo a su tenant", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.exec(t, a.id, "INSERT INTO sales_settings (tenant_id, max_discount) VALUES ($1, 0.1)", a.id)

		p, err := f.policies.Get(t.Context(), a.id)
		require.NoError(t, err)
		require.True(t, dec("0.1").Equal(p.MaxDiscount.Max()))

		p, err = f.policies.Get(t.Context(), b.id)
		require.NoError(t, err)
		require.True(t, p.MaxDiscount.Max().IsZero())
	})
}
