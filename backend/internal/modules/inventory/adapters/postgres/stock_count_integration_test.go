//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestStockCountStart(t *testing.T) {
	f := newFixture(t)

	t.Run("toma la foto de los niveles de la sede con el stock al inicio según el kardex", func(t *testing.T) {
		l := f.countedLocation(t)
		elsewhere := f.createLocation(t, l.tenant, "Bodega").ID()
		f.receive(t, l.tenant, f.createVariant(t, l.tenant), elsewhere, "5", "100")
		f.adjustAt(t, l.tenant, l.cable, l.location, "-5", now.Add(2*time.Minute))

		c, lines := f.startCount(t, l.tenant, l.location, now.Add(time.Minute))
		require.Equal(t, 2, lines)
		sheet := f.sheet(t, c)
		require.Len(t, sheet, 2)
		require.True(t, dec("100").Equal(sheet[l.cable].StartQuantity), "el ajuste es posterior al inicio")
		require.True(t, dec("2500").Equal(sheet[l.cable].StartCost))
		require.Equal(t, 1, sheet[l.cable].Round)
		require.Empty(t, sheet[l.cable].Counts)
		require.True(t, sheet[l.cable].Movements.IsZero())

		got, err := f.counts.Get(t.Context(), l.tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountOpen, got.Status())
		require.Equal(t, now.Add(time.Minute), got.StartedAt())
	})

	t.Run("el conteo parcial toma las variantes dadas, con o sin stock", func(t *testing.T) {
		l := f.countedLocation(t)
		gloves := f.createVariant(t, l.tenant)
		c, lines := f.startCount(t, l.tenant, l.location, now.Add(time.Minute), l.shirt, gloves, gloves)
		require.Equal(t, 2, lines)
		sheet := f.sheet(t, c)
		require.True(t, dec("10").Equal(sheet[l.shirt].StartQuantity))
		require.True(t, sheet[gloves].StartQuantity.IsZero())
		require.True(t, sheet[gloves].StartCost.IsZero())
	})

	t.Run("rechaza una segunda sesión abierta en la sede", func(t *testing.T) {
		l := f.countedLocation(t)
		f.startCount(t, l.tenant, l.location, now)
		c, err := domain.NewStockCount(domain.StockCountSpec{
			ID: newID(), TenantID: l.tenant, LocationID: l.location, Mode: domain.CountSum, StartedBy: newID(), StartedAt: now,
		})
		require.NoError(t, err)
		_, err = f.counts.Start(t.Context(), c, nil)
		require.ErrorIs(t, err, domain.ErrCountInProgress)
	})

	t.Run("rechaza una sede que no existe", func(t *testing.T) {
		tenant := f.createTenant(t)
		c, err := domain.NewStockCount(domain.StockCountSpec{
			ID: newID(), TenantID: tenant, LocationID: newID(), Mode: domain.CountSum, StartedBy: newID(), StartedAt: now,
		})
		require.NoError(t, err)
		_, err = f.counts.Start(t.Context(), c, nil)
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})
}

func TestStockCountSheet(t *testing.T) {
	f := newFixture(t)

	t.Run("la diferencia descuenta las ventas ocurridas durante el conteo, hasta el último conteo", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now.Add(time.Minute))
		f.adjustAt(t, l.tenant, l.cable, l.location, "-12.5", now.Add(2*time.Minute))
		ana, beto := newID(), newID()
		f.count(t, c, l.cable, ana, "50", now.Add(3*time.Minute))
		f.count(t, c, l.cable, beto, "20", now.Add(3*time.Minute))
		f.count(t, c, l.cable, ana, "15", now.Add(4*time.Minute))
		f.adjustAt(t, l.tenant, l.cable, l.location, "-3", now.Add(5*time.Minute))

		line := f.sheet(t, c)[l.cable]
		require.True(t, dec("-12.5").Equal(line.Movements), line.Movements.String())
		require.Len(t, line.Counts, 2)
		totals := map[uuid.UUID]string{}
		for _, ct := range line.Counts {
			totals[ct.UserID] = ct.Quantity.String()
		}
		require.Equal(t, map[uuid.UUID]string{ana: "65", beto: "20"}, totals)

		r := domain.EvaluateLine(line, domain.CountSum)
		require.True(t, dec("87.5").Equal(r.Expected))
		require.True(t, dec("-2.5").Equal(r.Difference))

		progress, err := f.counts.Progress(t.Context(), l.tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountProgress{Lines: 2, Counted: 1}, progress)
	})

	t.Run("un recuento deja fuera lo contado antes", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		f.count(t, c, l.shirt, newID(), "8", now.Add(time.Minute))

		require.NoError(t, f.counts.Recount(t.Context(), l.tenant, c.ID(), []uuid.UUID{l.shirt}))
		line := f.sheet(t, c)[l.shirt]
		require.Equal(t, 2, line.Round)
		require.Empty(t, line.Counts)

		f.count(t, c, l.shirt, newID(), "10", now.Add(2*time.Minute))
		require.True(t, dec("10").Equal(f.sheet(t, c)[l.shirt].Counts[0].Quantity))
	})

	t.Run("rechaza recontar una variante que no está, sin recontar las demás", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		err := f.counts.Recount(t.Context(), l.tenant, c.ID(), []uuid.UUID{l.shirt, newID()})
		require.ErrorIs(t, err, domain.ErrVariantNotInCount)
		require.Equal(t, 1, f.sheet(t, c)[l.shirt].Round)
	})
}

func TestStockCountClose(t *testing.T) {
	f := newFixture(t)

	t.Run("aprobar guarda el resumen y ajusta el stock en la misma transacción", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		f.count(t, c, l.cable, newID(), "98", now.Add(time.Minute))
		approver := newID()

		err := f.counts.Close(t.Context(), l.tenant, c.ID(), func(ctx context.Context, c *domain.StockCount) error {
			lines, err := f.counts.Sheet(ctx, c)
			require.NoError(t, err)
			var results []domain.LineResult
			for _, line := range lines {
				results = append(results, domain.EvaluateLine(line, c.Mode()))
			}
			adjustments, err := c.Approve(results, approver, now.Add(time.Hour))
			require.NoError(t, err)
			require.Len(t, adjustments, 1)
			return f.stock.Apply(ctx, l.tenant, l.cable, []uuid.UUID{l.location},
				func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
					m, err := levels[0].Adjust(adjustments[0].Delta, domain.CountAdjustmentReason, domain.StockPolicy{},
						c.AdjustmentInfo(approver, now.Add(time.Hour)))
					return []*domain.StockMovement{m}, err
				})
		})
		require.NoError(t, err)

		got, err := f.counts.Get(t.Context(), l.tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountApproved, got.Status())
		require.Equal(t, approver, got.ClosedBy())
		summary, ok := got.Summary()
		require.True(t, ok)
		require.Equal(t, 2, summary.Lines)
		require.Equal(t, 1, summary.Counted)
		require.True(t, dec("-5000").Equal(summary.NetValue))
		require.True(t, dec("98").Equal(f.level(t, l.tenant, l.cable, l.location)))

		moves, err := f.stock.MovementsByReference(t.Context(), l.tenant, domain.DocumentRef{Kind: domain.DocumentStockCount, ID: c.ID()})
		require.NoError(t, err)
		require.Len(t, moves, 1)
		require.Equal(t, domain.CountAdjustmentReason, moves[0].Reason())

		e, err := got.NewEntry(newID(), l.cable, dec("1"), newID(), now)
		require.ErrorIs(t, err, domain.ErrCountNotOpen, "el dominio ya no deja contar")
		require.Nil(t, e)
	})

	t.Run("aprobar es atómico: si algo falla, ni el stock ni el conteo cambian", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		boom := errors.New("boom")

		err := f.counts.Close(t.Context(), l.tenant, c.ID(), func(ctx context.Context, c *domain.StockCount) error {
			_, err := c.Approve(nil, newID(), now)
			require.NoError(t, err)
			f.adjustIn(ctx, t, l, "7")
			return boom
		})
		require.ErrorIs(t, err, boom)
		require.True(t, dec("100").Equal(f.level(t, l.tenant, l.cable, l.location)))
		got, err := f.counts.Get(t.Context(), l.tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountOpen, got.Status())
	})

	t.Run("cancelar cierra el conteo y libera la sede", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		require.NoError(t, f.counts.Close(t.Context(), l.tenant, c.ID(), func(_ context.Context, c *domain.StockCount) error {
			return c.Cancel(newID(), now)
		}))
		e, err := c.NewEntry(newID(), l.cable, dec("1"), newID(), now)
		require.NoError(t, err, "la copia en memoria sigue abierta")
		require.ErrorIs(t, f.entries.Add(t.Context(), e, true), domain.ErrCountNotOpen)
		f.startCount(t, l.tenant, l.location, now.Add(time.Hour))
	})

	t.Run("un conteo que no existe es ErrCountNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		err := f.counts.Close(t.Context(), tenant, newID(), func(context.Context, *domain.StockCount) error { return nil })
		require.ErrorIs(t, err, domain.ErrCountNotFound)
	})
}

// adjustIn adds qty of the cable inside the transaction ctx carries.
func (f fixture) adjustIn(ctx context.Context, t *testing.T, l countedLocation, qty string) {
	t.Helper()
	require.NoError(t, f.stock.Apply(ctx, l.tenant, l.cable, []uuid.UUID{l.location},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			m, err := levels[0].Adjust(dec(qty), "Prueba", domain.StockPolicy{}, info(newID()))
			return []*domain.StockMovement{m}, err
		}))
}
