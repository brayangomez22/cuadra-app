package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestCustomerSegment(t *testing.T) {
	t.Run("crea segmentos de distintos tipos de negocio con el nombre sin espacios sobrantes", func(t *testing.T) {
		for _, name := range []string{"  Maestro de obra ", "Mayorista", "Convenio EPS"} {
			s, err := domain.NewSegment(tenantID, name, now)
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(name), s.Name())
			require.Equal(t, tenantID, s.TenantID())
			require.NotEqual(t, tenantID, s.ID())
		}
	})

	t.Run("rechaza segmento sin nombre o con nombre demasiado largo", func(t *testing.T) {
		for _, name := range []string{"", "   ", strings.Repeat("a", 61)} {
			_, err := domain.NewSegment(tenantID, name, now)
			require.ErrorIs(t, err, domain.ErrInvalidSegmentName)
		}
	})

	t.Run("renombrar con un nombre inválido deja el segmento igual", func(t *testing.T) {
		s, err := domain.NewSegment(tenantID, "Mayorista", now)
		require.NoError(t, err)
		require.ErrorIs(t, s.Rename(" "), domain.ErrInvalidSegmentName)
		require.Equal(t, "Mayorista", s.Name())
		require.NoError(t, s.Rename("Mayorista VIP"))
		require.Equal(t, "Mayorista VIP", s.Name())
	})
}
