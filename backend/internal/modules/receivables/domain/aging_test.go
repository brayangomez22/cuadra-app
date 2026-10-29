package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestDate(t *testing.T) {
	t.Run("toma el día calendario de America/Bogota", func(t *testing.T) {
		cases := []struct {
			name string
			at   time.Time
			want domain.Date
		}{
			{"media mañana", time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC), domain.NewDate(2026, 10, 28)},
			{"21:00 en Bogotá ya es otro día en UTC", time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC), domain.NewDate(2026, 10, 28)},
			{"medianoche en Bogotá", time.Date(2026, 10, 29, 5, 0, 0, 0, time.UTC), domain.NewDate(2026, 10, 29)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.want, domain.DateIn(c.at))
			})
		}
	})

	t.Run("suma días y cuenta los días entre dos fechas", func(t *testing.T) {
		d := domain.NewDate(2026, 12, 20)
		require.Equal(t, domain.NewDate(2027, 1, 19), d.AddDays(30))
		require.Equal(t, 30, domain.NewDate(2027, 1, 19).DaysSince(d))
		require.Equal(t, -30, d.DaysSince(domain.NewDate(2027, 1, 19)))
		require.Equal(t, "2026-12-20", d.String())
	})
}

func TestAgingBuckets(t *testing.T) {
	due := domain.NewDate(2026, 9, 1)
	cases := []struct {
		name  string
		today domain.Date
		want  domain.AgingBucket
	}{
		{"un día antes del vencimiento está por vencer", domain.NewDate(2026, 8, 31), domain.AgingCurrent},
		{"el día del vencimiento todavía está por vencer", due, domain.AgingCurrent},
		{"un día vencida", domain.NewDate(2026, 9, 2), domain.AgingDays1To30},
		{"30 días vencida", domain.NewDate(2026, 10, 1), domain.AgingDays1To30},
		{"31 días vencida", domain.NewDate(2026, 10, 2), domain.AgingDays31To60},
		{"60 días vencida", domain.NewDate(2026, 10, 31), domain.AgingDays31To60},
		{"61 días vencida", domain.NewDate(2026, 11, 1), domain.AgingDays61To90},
		{"90 días vencida", domain.NewDate(2026, 11, 30), domain.AgingDays61To90},
		{"91 días vencida", domain.NewDate(2026, 12, 1), domain.AgingOver90},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, domain.BucketOf(due, c.today))
		})
	}
}

func TestAging(t *testing.T) {
	t.Run("suma los saldos por edad con la fecha de vencimiento en America/Bogota", func(t *testing.T) {
		// Today is 2026-10-28 in Bogotá although it is already the 29th in UTC.
		today := domain.DateIn(time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC))
		g := domain.NewAging(today)
		g.Add(domain.NewDate(2026, 10, 28), dec("100000"))   // ferretería: vence hoy
		g.Add(domain.NewDate(2026, 10, 27), dec("25000.50")) // almacén de ropa: un día
		g.Add(domain.NewDate(2026, 9, 1), dec("40000"))      // 57 días
		g.Add(domain.NewDate(2026, 6, 1), dec("10000"))      // 149 días
		require.True(t, dec("100000").Equal(g.Current), g.Current.String())
		require.True(t, dec("25000.50").Equal(g.Days1To30), g.Days1To30.String())
		require.True(t, dec("40000").Equal(g.Days31To60), g.Days31To60.String())
		require.True(t, g.Days61To90.IsZero())
		require.True(t, dec("10000").Equal(g.Over90), g.Over90.String())
		require.True(t, dec("75000.50").Equal(g.Overdue()))
		require.True(t, dec("175000.50").Equal(g.Total()))
	})
}
