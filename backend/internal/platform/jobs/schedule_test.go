package jobs_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

func TestDaily(t *testing.T) {
	bogota, err := time.LoadLocation("America/Bogota")
	require.NoError(t, err)
	at3 := jobs.Daily(3, 0, bogota)

	cases := []struct {
		name    string
		current time.Time
		want    time.Time
	}{
		{"después de las 3:00 corre a las 3:00 de Bogotá del día siguiente",
			time.Date(2026, 10, 28, 15, 0, 0, 0, bogota), time.Date(2026, 10, 29, 3, 0, 0, 0, bogota)},
		{"antes de las 3:00 corre ese mismo día",
			time.Date(2026, 10, 28, 1, 30, 0, 0, bogota), time.Date(2026, 10, 28, 3, 0, 0, 0, bogota)},
		{"justo a las 3:00 corre al día siguiente",
			time.Date(2026, 10, 28, 3, 0, 0, 0, bogota), time.Date(2026, 10, 29, 3, 0, 0, 0, bogota)},
		{"recibe la hora en UTC y responde la de Bogotá",
			time.Date(2026, 12, 31, 9, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 3, 0, 0, 0, bogota)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.True(t, c.want.Equal(at3.Next(c.current)), "got %s", at3.Next(c.current))
		})
	}
}
