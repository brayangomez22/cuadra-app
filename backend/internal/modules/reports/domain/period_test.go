package domain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func day(m time.Month, d int) calendar.Date { return calendar.NewDate(2026, m, d) }

func TestNewPeriod(t *testing.T) {
	tests := []struct {
		name     string
		from, to calendar.Date
		wantErr  error
	}{
		{"acepta un solo día", day(time.November, 3), day(time.November, 3), nil},
		{"acepta un año bisiesto completo", calendar.NewDate(2028, time.January, 1), calendar.NewDate(2028, time.December, 31), nil},
		{"rechaza un periodo con la fecha final antes de la inicial", day(time.November, 3), day(time.November, 2), domain.ErrInvalidPeriod},
		{"rechaza un periodo de más de 366 días", day(time.January, 1), calendar.NewDate(2027, time.January, 2), domain.ErrInvalidPeriod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := domain.NewPeriod(tt.from, tt.to)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				require.Equal(t, tt.from, p.From())
				require.Equal(t, tt.to, p.To())
			}
		})
	}

	t.Run("va de la medianoche de Colombia del primer día a la del día siguiente al último", func(t *testing.T) {
		p, err := domain.NewPeriod(day(time.November, 3), day(time.November, 4))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, time.November, 3, 5, 0, 0, 0, time.UTC), p.Start().UTC())
		require.Equal(t, time.Date(2026, time.November, 5, 5, 0, 0, 0, time.UTC), p.End().UTC())
	})
	t.Run("una venta a las 11 de la noche cuenta en ese día y no en el siguiente", func(t *testing.T) {
		p, err := domain.NewPeriod(day(time.November, 3), day(time.November, 3))
		require.NoError(t, err)
		at := time.Date(2026, time.November, 4, 4, 0, 0, 0, time.UTC) // 11 p. m. del 3 en Bogotá
		require.False(t, at.Before(p.Start()))
		require.True(t, at.Before(p.End()))
	})
}

func TestNewSince(t *testing.T) {
	today := day(time.November, 19)
	t.Run("es el día N días antes de hoy", func(t *testing.T) {
		s, err := domain.NewSince(today, 30)
		require.NoError(t, err)
		require.Equal(t, day(time.October, 20), s.Day)
		require.Equal(t, 30, s.Days)
	})
	for _, days := range []int{0, -1, domain.MaxIdleDays + 1} {
		t.Run(fmt.Sprintf("rechaza %d días", days), func(t *testing.T) {
			_, err := domain.NewSince(today, days)
			require.ErrorIs(t, err, domain.ErrInvalidFilter)
		})
	}
}

func TestOptions(t *testing.T) {
	t.Run("un agrupamiento vacío toma el de por defecto", func(t *testing.T) {
		g, err := domain.ParseSalesGroup("")
		require.NoError(t, err)
		require.Equal(t, domain.SalesByDay, g)
		b, err := domain.ParseBucket("")
		require.NoError(t, err)
		require.Equal(t, domain.BucketWeek, b)
	})
	t.Run("rechaza un agrupamiento que el reporte no tiene", func(t *testing.T) {
		_, err := domain.ParseCashGroup("payment_method")
		require.ErrorIs(t, err, domain.ErrInvalidFilter)
		_, err = domain.ParseAdjustmentGroup("day")
		require.ErrorIs(t, err, domain.ErrInvalidFilter)
	})
	t.Run("un filtro por atributo exige la clave y el valor", func(t *testing.T) {
		key, value := "marca", "Acme"
		_, err := domain.NewProductScope(nil, &key, nil)
		require.ErrorIs(t, err, domain.ErrInvalidFilter)
		s, err := domain.NewProductScope(nil, &key, &value)
		require.NoError(t, err)
		require.Equal(t, "Acme", s.AttributeValue)
	})
}
