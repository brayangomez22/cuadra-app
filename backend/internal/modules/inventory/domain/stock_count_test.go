package domain_test

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func countSpec(mode domain.CountMode, categories ...uuid.UUID) domain.StockCountSpec {
	return domain.StockCountSpec{
		ID: newID(), TenantID: newID(), LocationID: newID(), CategoryIDs: categories,
		Mode: mode, StartedBy: newID(), StartedAt: testNow,
	}
}

func openCount(t *testing.T, mode domain.CountMode) *domain.StockCount {
	t.Helper()
	c, err := domain.NewStockCount(countSpec(mode))
	require.NoError(t, err)
	return c
}

func TestNewStockCount(t *testing.T) {
	t.Run("crea un conteo total abierto", func(t *testing.T) {
		s := countSpec(domain.CountSum)
		c, err := domain.NewStockCount(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, c.ID())
		require.Equal(t, s.TenantID, c.TenantID())
		require.Equal(t, s.LocationID, c.LocationID())
		require.Equal(t, domain.CountOpen, c.Status())
		require.Equal(t, domain.CountSum, c.Mode())
		require.False(t, c.IsPartial())
		require.Equal(t, s.StartedBy, c.StartedBy())
		require.Equal(t, testNow.UTC(), c.StartedAt())
		require.NoError(t, c.CheckOpen())
	})

	t.Run("crea un conteo parcial por categorías sin repetirlas", func(t *testing.T) {
		tools, shirts := newID(), newID()
		c, err := domain.NewStockCount(countSpec(domain.CountReview, tools, shirts, tools))
		require.NoError(t, err)
		require.True(t, c.IsPartial())
		require.ElementsMatch(t, []uuid.UUID{tools, shirts}, c.CategoryIDs())
	})

	invalid := []struct {
		name string
		edit func(*domain.StockCountSpec)
		err  error
	}{
		{"rechaza un conteo sin id", func(s *domain.StockCountSpec) { s.ID = uuid.Nil }, domain.ErrInvalidCountID},
		{"rechaza un conteo sin tenant", func(s *domain.StockCountSpec) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
		{"rechaza un conteo sin sede", func(s *domain.StockCountSpec) { s.LocationID = uuid.Nil }, domain.ErrInvalidLocationID},
		{"rechaza un conteo sin usuario", func(s *domain.StockCountSpec) { s.StartedBy = uuid.Nil }, domain.ErrInvalidUserID},
		{"rechaza un modo desconocido", func(s *domain.StockCountSpec) { s.Mode = "max" }, domain.ErrInvalidCountMode},
		{"rechaza una categoría vacía", func(s *domain.StockCountSpec) { s.CategoryIDs = []uuid.UUID{uuid.Nil} }, domain.ErrInvalidCategoryID},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			s := countSpec(domain.CountSum)
			tc.edit(&s)
			_, err := domain.NewStockCount(s)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestStockCountNewEntry(t *testing.T) {
	c := openCount(t, domain.CountSum)
	user := newID()

	for _, qty := range []string{"12.5", "3", "0"} {
		t.Run("registra lo contado: "+qty, func(t *testing.T) {
			id, variant := newID(), newID()
			e, err := c.NewEntry(id, variant, dec(qty), user, testNow)
			require.NoError(t, err)
			require.Equal(t, id, e.ID())
			require.Equal(t, c.TenantID(), e.TenantID())
			require.Equal(t, c.ID(), e.CountID())
			require.Equal(t, variant, e.VariantID())
			requireDec(t, qty, e.Quantity())
			require.Equal(t, user, e.CountedBy())
			require.Equal(t, testNow.UTC(), e.CountedAt())
		})
	}

	for _, qty := range []string{"-1", "1.00001", "100000000000000"} {
		t.Run("rechaza un conteo negativo, con más de 4 decimales o que no cabe: "+qty, func(t *testing.T) {
			_, err := c.NewEntry(newID(), newID(), dec(qty), user, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		})
	}

	t.Run("rechaza conteos en una sesión aprobada o cancelada", func(t *testing.T) {
		approved := openCount(t, domain.CountSum)
		_, err := approved.Approve(nil, user, testNow)
		require.NoError(t, err)
		canceled := openCount(t, domain.CountSum)
		require.NoError(t, canceled.Cancel(user, testNow))

		for _, closed := range []*domain.StockCount{approved, canceled} {
			_, err := closed.NewEntry(newID(), newID(), dec("1"), user, testNow)
			require.ErrorIs(t, err, domain.ErrCountNotOpen)
		}
	})
}

func TestStockCountRemoveEntry(t *testing.T) {
	c := openCount(t, domain.CountSum)
	counter := newID()
	e, err := c.NewEntry(newID(), newID(), dec("4"), counter, testNow)
	require.NoError(t, err)

	t.Run("quien contó puede borrar su conteo", func(t *testing.T) {
		require.NoError(t, c.CheckEntryRemoval(e, counter, false))
	})
	t.Run("quien administra puede borrar el conteo de otro", func(t *testing.T) {
		require.NoError(t, c.CheckEntryRemoval(e, newID(), true))
	})
	t.Run("solo borra el conteo propio salvo quien administra", func(t *testing.T) {
		require.ErrorIs(t, c.CheckEntryRemoval(e, newID(), false), domain.ErrCountEntryNotYours)
	})
	t.Run("no borra conteos de una sesión cerrada", func(t *testing.T) {
		require.NoError(t, c.Cancel(counter, testNow))
		require.ErrorIs(t, c.CheckEntryRemoval(e, counter, true), domain.ErrCountNotOpen)
	})
}

func TestStockCountApprove(t *testing.T) {
	approver := newID()
	at := testNow.Add(2 * time.Hour)

	t.Run("la aprobación solo genera ajustes para las líneas con diferencia", func(t *testing.T) {
		c := openCount(t, domain.CountSum)
		short := sheetLine("10", "1000", "-1", "8")
		extra := sheetLine("4", "2500", "0", "5.5")
		exact := sheetLine("6", "100", "0", "6")
		uncounted := sheetLine("9", "100", "0")

		adjustments, err := c.Approve(results(short, extra, exact, uncounted), approver, at)
		require.NoError(t, err)

		want := []uuid.UUID{short.VariantID, extra.VariantID}
		slices.SortFunc(want, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
		require.Equal(t, want, variantsOf(adjustments), "en orden de variante")
		for _, a := range adjustments {
			if a.VariantID == short.VariantID {
				requireDec(t, "-1", a.Delta)
			} else {
				requireDec(t, "1.5", a.Delta)
			}
		}

		require.Equal(t, domain.CountApproved, c.Status())
		require.Equal(t, approver, c.ClosedBy())
		require.Equal(t, at.UTC(), c.ClosedAt())
		summary, ok := c.Summary()
		require.True(t, ok)
		require.Equal(t, 3, summary.Counted)
		require.Equal(t, 1, summary.Exact)
	})

	t.Run("no aprueba con variantes en revisión", func(t *testing.T) {
		c := openCount(t, domain.CountReview)
		disputed := domain.EvaluateLine(sheetLine("5", "100", "0", "4", "5"), domain.CountReview)
		_, err := c.Approve([]domain.LineResult{disputed}, approver, at)
		require.ErrorIs(t, err, domain.ErrCountNeedsReview)
		require.Equal(t, domain.CountOpen, c.Status())
	})

	t.Run("no aprueba ni cancela una sesión cerrada", func(t *testing.T) {
		c := openCount(t, domain.CountSum)
		require.NoError(t, c.Cancel(approver, at))
		_, err := c.Approve(nil, approver, at)
		require.ErrorIs(t, err, domain.ErrCountNotOpen)
		require.ErrorIs(t, c.Cancel(approver, at), domain.ErrCountNotOpen)
		_, ok := c.Summary()
		require.False(t, ok)
	})

	t.Run("los ajustes llevan el motivo conteo físico y la referencia a la sesión", func(t *testing.T) {
		c := openCount(t, domain.CountSum)
		info := c.AdjustmentInfo(approver, at)
		require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentStockCount, ID: c.ID()}, info.Reference)
		require.Equal(t, approver, info.UserID)

		level := levelWith(t, "10", "1000")
		m, err := level.Adjust(dec("-2"), domain.CountAdjustmentReason, strict, info)
		require.NoError(t, err)
		require.Equal(t, "Conteo físico", m.Reason())
		require.Equal(t, c.ID(), m.Reference().ID)
	})
}

func TestRehydrateStockCount(t *testing.T) {
	t.Run("reconstruye un conteo aprobado con su resumen", func(t *testing.T) {
		summary := domain.CountSummary{Lines: 3, Counted: 2, Exact: 1, NetValue: dec("-100"), AbsoluteValue: dec("300")}
		c, err := domain.RehydrateStockCount(domain.StockCountSnapshot{
			StockCountSpec: countSpec(domain.CountSum, newID()),
			Status:         domain.CountApproved, ClosedBy: newID(), ClosedAt: testNow, Summary: &summary,
		})
		require.NoError(t, err)
		got, ok := c.Summary()
		require.True(t, ok)
		require.Equal(t, summary, got)
		require.True(t, c.IsPartial())
	})

	t.Run("rechaza un conteo cerrado sin quién ni cuándo", func(t *testing.T) {
		_, err := domain.RehydrateStockCount(domain.StockCountSnapshot{
			StockCountSpec: countSpec(domain.CountSum), Status: domain.CountCanceled,
		})
		require.ErrorIs(t, err, domain.ErrInvalidStockCount)
	})
}
