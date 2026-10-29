package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

func TestDate(t *testing.T) {
	t.Run("el día en America/Bogota es el anterior antes de las 5 de la mañana UTC", func(t *testing.T) {
		at := time.Date(2026, time.March, 2, 4, 59, 0, 0, time.UTC)
		require.Equal(t, calendar.NewDate(2026, time.March, 1), calendar.DateIn(at))
	})
	t.Run("suma días y cuenta los días entre dos fechas", func(t *testing.T) {
		d := calendar.NewDate(2026, time.January, 30)
		require.Equal(t, calendar.NewDate(2026, time.March, 1), d.AddDays(30))
		require.Equal(t, 30, d.AddDays(30).DaysSince(d))
	})
	t.Run("una fecha es anterior a otra solo si es de un día antes", func(t *testing.T) {
		d := calendar.NewDate(2026, time.May, 10)
		require.True(t, d.Before(d.AddDays(1)))
		require.False(t, d.Before(d))
	})
}
