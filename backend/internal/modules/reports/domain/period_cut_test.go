package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func TestPeriodUntil(t *testing.T) {
	week, err := domain.NewPeriod(day(time.November, 12), day(time.November, 19))
	require.NoError(t, err)

	t.Run("con corte de hora, el último día termina a esa hora de Colombia", func(t *testing.T) {
		p, err := week.Until("10:30")
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, time.November, 19, 15, 30, 0, 0, time.UTC), p.End().UTC())
	})
	t.Run("el corte no cambia el inicio ni los días del periodo", func(t *testing.T) {
		p, err := week.Until("10:30")
		require.NoError(t, err)
		require.Equal(t, week.Start(), p.Start())
		require.Equal(t, week.From(), p.From())
		require.Equal(t, week.To(), p.To())
	})
	t.Run("a la medianoche el último día no cuenta", func(t *testing.T) {
		p, err := week.Until("00:00")
		require.NoError(t, err)
		require.Equal(t, day(time.November, 19).Start(), p.End())
	})
	t.Run("sin corte el periodo no cambia", func(t *testing.T) {
		p, err := week.Until("")
		require.NoError(t, err)
		require.Equal(t, week.End(), p.End())
	})
	for _, clock := range []string{"24:00", "9:30", "10:60", "10:30:00", "mediodía"} {
		t.Run("rechaza la hora de corte "+clock, func(t *testing.T) {
			_, err := week.Until(clock)
			require.ErrorIs(t, err, domain.ErrInvalidPeriod)
		})
	}
}
