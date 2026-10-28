package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// sheetLine is a line that started with start units at cost, with the
// movements of the kardex during the count and what each counter counted.
func sheetLine(start, cost, movements string, counts ...string) domain.SheetLine {
	l := domain.SheetLine{
		CountLine: domain.CountLine{VariantID: newID(), StartQuantity: dec(start), StartCost: dec(cost), Round: 1},
		Movements: dec(movements),
	}
	for _, c := range counts {
		l.Counts = append(l.Counts, domain.CounterTotal{UserID: newID(), Quantity: dec(c)})
	}
	return l
}

func TestEvaluateLine(t *testing.T) {
	cases := []struct {
		name                         string
		line                         domain.SheetLine
		mode                         domain.CountMode
		status                       domain.LineStatus
		expected, counted, diff, val string
	}{
		{
			"calcula la diferencia como contado menos el stock al inicio más entradas menos salidas durante el conteo (metros de cable)",
			sheetLine("100", "2500", "-12.5", "85"), domain.CountSum,
			domain.LineDifference, "87.5", "85", "-2.5", "-6250",
		},
		{
			"una entrada durante el conteo sube lo esperado (camisetas talla M)",
			sheetLine("10", "18000", "6", "16"), domain.CountSum,
			domain.LineExact, "16", "16", "0", "0",
		},
		{
			"valora la diferencia al costo del inicio con redondeo half-up",
			sheetLine("3", "1000.005", "0", "4"), domain.CountSum,
			domain.LineDifference, "3", "4", "1", "1000.01",
		},
		{
			"suma los conteos de dos contadores sobre la misma variante en modo suma",
			sheetLine("20", "500", "0", "12", "8"), domain.CountSum,
			domain.LineExact, "20", "20", "0", "0",
		},
		{
			"marca para revisión si dos contadores difieren en modo revisión",
			sheetLine("20", "500", "0", "12", "8"), domain.CountReview,
			domain.LineReview, "20", "0", "0", "0",
		},
		{
			"no marca revisión si dos contadores coinciden en modo revisión",
			sheetLine("6", "45000", "-1", "4", "4"), domain.CountReview,
			domain.LineDifference, "5", "4", "-1", "-45000",
		},
		{
			"una línea sin conteos queda sin contar y sin diferencia",
			sheetLine("7", "300", "-2"), domain.CountSum,
			domain.LineUncounted, "5", "0", "0", "0",
		},
		{
			"contar cero en una variante con stock es una diferencia",
			sheetLine("2", "9900", "0", "0"), domain.CountReview,
			domain.LineDifference, "2", "0", "-2", "-19800",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := domain.EvaluateLine(tc.line, tc.mode)
			require.Equal(t, tc.line.VariantID, r.VariantID)
			require.Equal(t, tc.status, r.Status)
			requireDec(t, tc.expected, r.Expected)
			requireDec(t, tc.counted, r.Counted)
			requireDec(t, tc.diff, r.Difference)
			requireDec(t, tc.val, r.Value)
		})
	}
}

func TestSummarize(t *testing.T) {
	results := []domain.LineResult{
		domain.EvaluateLine(sheetLine("10", "1000", "0", "10"), domain.CountSum),
		domain.EvaluateLine(sheetLine("10", "1000", "0", "8"), domain.CountSum),
		domain.EvaluateLine(sheetLine("4", "2500", "0", "5"), domain.CountSum),
		domain.EvaluateLine(sheetLine("4", "2500", "0", "3"), domain.CountSum),
		domain.EvaluateLine(sheetLine("1", "1", "0"), domain.CountSum),
	}

	t.Run("calcula la exactitud del conteo", func(t *testing.T) {
		s := domain.Summarize(results)
		require.Equal(t, 5, s.Lines)
		require.Equal(t, 4, s.Counted)
		require.Equal(t, 1, s.Exact)
		requireDec(t, "-2000", s.NetValue)
		requireDec(t, "7000", s.AbsoluteValue)
		accuracy, ok := s.Accuracy()
		require.True(t, ok)
		requireDec(t, "0.25", accuracy)
	})

	t.Run("sin líneas contadas no hay exactitud", func(t *testing.T) {
		_, ok := domain.Summarize(results[4:]).Accuracy()
		require.False(t, ok)
	})
}

func TestParseCountFilters(t *testing.T) {
	t.Run("acepta los modos y estados de un conteo", func(t *testing.T) {
		for _, s := range []string{"sum", "review"} {
			m, err := domain.ParseCountMode(s)
			require.NoError(t, err)
			require.Equal(t, s, string(m))
		}
		for _, s := range []string{"open", "approved", "canceled"} {
			st, err := domain.ParseCountStatus(s)
			require.NoError(t, err)
			require.Equal(t, s, string(st))
		}
		for _, s := range []string{"uncounted", "exact", "difference", "review"} {
			st, err := domain.ParseLineStatus(s)
			require.NoError(t, err)
			require.Equal(t, s, string(st))
		}
	})

	t.Run("rechaza valores desconocidos", func(t *testing.T) {
		_, err := domain.ParseCountMode("SUM")
		require.ErrorIs(t, err, domain.ErrInvalidCountMode)
		_, err = domain.ParseCountStatus("closed")
		require.ErrorIs(t, err, domain.ErrInvalidCountStatus)
		_, err = domain.ParseLineStatus("")
		require.ErrorIs(t, err, domain.ErrInvalidLineStatus)
	})
}

// results evaluates lines of different variants in sum mode.
func results(lines ...domain.SheetLine) []domain.LineResult {
	out := make([]domain.LineResult, 0, len(lines))
	for _, l := range lines {
		out = append(out, domain.EvaluateLine(l, domain.CountSum))
	}
	return out
}

func variantsOf(adjustments []domain.CountAdjustment) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(adjustments))
	for _, a := range adjustments {
		ids = append(ids, a.VariantID)
	}
	return ids
}
