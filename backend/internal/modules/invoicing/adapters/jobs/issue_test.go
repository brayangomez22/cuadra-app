package jobs_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/jobs"
)

func TestBackoff(t *testing.T) {
	cases := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"el primer reintento espera 10 segundos", 1, 10 * time.Second},
		{"cada reintento dobla la espera", 4, 80 * time.Second},
		{"la espera no pasa de una hora", 12, time.Hour},
		{"un intento enorme no desborda la espera", 1000, time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, jobs.Backoff(tc.attempt))
		})
	}
}
