//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// adjustAt changes the stock of a variant by delta at a moment, like a sale
// or a receipt during a count.
func (f fixture) adjustAt(t *testing.T, tenantID, variantID, locationID uuid.UUID, delta string, at time.Time) {
	t.Helper()
	err := f.stock.Apply(t.Context(), tenantID, variantID, []uuid.UUID{locationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			m, err := levels[0].Adjust(dec(delta), "Venta de mostrador", domain.StockPolicy{AllowNegative: true},
				domain.MovementInfo{UserID: newID(), OccurredAt: at})
			return []*domain.StockMovement{m}, err
		})
	require.NoError(t, err)
}

// startCount opens a count at a location started at at: full without
// variants, partial over the variants given.
func (f fixture) startCount(t *testing.T, tenantID, locationID uuid.UUID, at time.Time, variants ...uuid.UUID) (*domain.StockCount, int) {
	t.Helper()
	spec := domain.StockCountSpec{
		ID: newID(), TenantID: tenantID, LocationID: locationID, Mode: domain.CountSum, StartedBy: newID(), StartedAt: at,
	}
	if len(variants) > 0 {
		spec.CategoryIDs = []uuid.UUID{newID()}
	}
	c, err := domain.NewStockCount(spec)
	require.NoError(t, err)
	lines, err := f.counts.Start(t.Context(), c, variants)
	require.NoError(t, err)
	return c, lines
}

// count records that user counted qty of a variant at a moment.
func (f fixture) count(t *testing.T, c *domain.StockCount, variantID, user uuid.UUID, qty string, at time.Time) *domain.CountEntry {
	t.Helper()
	e, err := c.NewEntry(newID(), variantID, dec(qty), user, at)
	require.NoError(t, err)
	require.NoError(t, f.entries.Add(t.Context(), e, !c.IsPartial()))
	return e
}

// sheet returns the lines of a count by variant.
func (f fixture) sheet(t *testing.T, c *domain.StockCount) map[uuid.UUID]domain.SheetLine {
	t.Helper()
	lines, err := f.counts.Sheet(t.Context(), c)
	require.NoError(t, err)
	out := map[uuid.UUID]domain.SheetLine{}
	for _, l := range lines {
		out[l.VariantID] = l
	}
	return out
}

// countedLocation is a tenant with a location holding 100 m of cable (at
// 2.500) and 10 T-shirts (at 18.000), received at now.
type countedLocation struct {
	tenant, location, cable, shirt uuid.UUID
}

func (f fixture) countedLocation(t *testing.T) countedLocation {
	t.Helper()
	tenant := f.createTenant(t)
	l := countedLocation{tenant: tenant, location: f.createLocation(t, tenant, "Principal").ID()}
	l.cable, l.shirt = f.createVariant(t, tenant), f.createVariant(t, tenant)
	f.receive(t, tenant, l.cable, l.location, "100", "2500")
	f.receive(t, tenant, l.shirt, l.location, "10", "18000")
	return l
}
