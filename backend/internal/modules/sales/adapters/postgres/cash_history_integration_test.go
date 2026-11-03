//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestCashSessionRepositorySearch(t *testing.T) {
	f := newFixture(t)
	repo := postgres.NewCashSessionRepository(f.db)

	t.Run("lista las cajas, la más reciente primero, con lo vendido de las cerradas", func(t *testing.T) {
		tn := f.createTenant(t)
		closed := openSession(t, repo, tn, "100000")
		f.save(t, saleIn(t, tn, closed))
		_, err := repo.Close(t.Context(), tn.id, closed.ID(), closeWith(closed, "100000", 2))
		require.NoError(t, err)
		open := openSession(t, repo, tn, "0")

		all, total, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: tn.id, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, all, 2)

		byID := map[string]domain.CashSessionSummary{}
		for _, s := range all {
			byID[s.ID.String()] = s
		}
		c := byID[closed.ID().String()]
		require.Equal(t, domain.CashSessionClosed, c.Status)
		require.Equal(t, 1, c.SalesCount)
		// The sale: 123760 (80000 in cash, net of 6240 of change, and 50000 by Nequi).
		require.True(t, dec("123760").Equal(c.SalesTotal), c.SalesTotal.String())
		require.NotNil(t, c.ClosedAt)
		require.Equal(t, closed.UserID(), *c.ClosedBy)
		result, ok := c.Result()
		require.True(t, ok)
		require.Equal(t, domain.CashOverage, result)

		o := byID[open.ID().String()]
		require.Equal(t, domain.CashSessionOpen, o.Status)
		_, ok = o.Result()
		require.False(t, ok)
		require.True(t, o.SalesTotal.IsZero())
	})

	t.Run("filtra por estado, sede y días de apertura", func(t *testing.T) {
		tn := f.createTenant(t)
		openSession(t, repo, tn, "0")
		otherLocation := newID()

		open, total, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: tn.id, Status: domain.CashSessionOpen, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, open, 1)

		none, _, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: tn.id, LocationID: &otherLocation, Limit: 10})
		require.NoError(t, err)
		require.Empty(t, none)

		after := now.Add(time.Minute)
		later, _, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: tn.id, OpenedFrom: &after, Limit: 10})
		require.NoError(t, err)
		require.Empty(t, later)
		until := now.Add(time.Minute)
		before, _, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: tn.id, OpenedBefore: &until, Limit: 10})
		require.NoError(t, err)
		require.Len(t, before, 1)
	})

	t.Run("aislamiento: otra empresa no ve las cajas", func(t *testing.T) {
		tn, other := f.createTenant(t), f.createTenant(t)
		openSession(t, repo, tn, "0")

		items, total, err := repo.Search(t.Context(), domain.CashSessionFilter{TenantID: other.id, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, total)
		require.Empty(t, items)
	})
}
