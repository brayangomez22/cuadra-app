package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestUnitDIANCode(t *testing.T) {
	t.Run("acepta el código DIAN de la unidad y lo guarda en mayúsculas", func(t *testing.T) {
		tests := []struct{ name, code, unitName, dian, want string }{
			{"ferretería: metro", "m", "Metro", " mtr ", "MTR"},
			{"almacén de calzado: par", "par", "Par", "PR", "PR"},
			{"tienda: unidad", "und", "Unidad", "94", "94"},
			{"sin código: la factura usa el de unidad", "caja", "Caja", "  ", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				u, err := domain.NewUnit(uuid.Must(uuid.NewV7()), tt.code, tt.unitName, testNow)
				require.NoError(t, err)

				require.NoError(t, u.SetDIANCode(tt.dian))

				require.Equal(t, tt.want, u.DIANCode())
				require.Equal(t, tt.want, u.Ref().DIANCode)
			})
		}
	})

	t.Run("rechaza un código DIAN con más de tres caracteres o con símbolos", func(t *testing.T) {
		for _, dian := range []string{"KGMS", "M-T", "m t", "ñ"} {
			u, err := domain.NewUnit(uuid.Must(uuid.NewV7()), "kg", "Kilogramo", testNow)
			require.NoError(t, err)
			require.NoError(t, u.SetDIANCode("KGM"))

			require.ErrorIs(t, u.SetDIANCode(dian), domain.ErrInvalidUnitDIANCode, "%q", dian)
			require.Equal(t, "KGM", u.DIANCode())
		}
	})

	t.Run("rehidrata el código DIAN guardado y rechaza uno inválido", func(t *testing.T) {
		s := domain.UnitSnapshot{
			ID: uuid.Must(uuid.NewV7()), TenantID: uuid.Must(uuid.NewV7()), Code: "kg", Name: "Kilogramo",
			DIANCode: "KGM", CreatedAt: testNow,
		}
		u, err := domain.RehydrateUnit(s)
		require.NoError(t, err)
		require.Equal(t, "KGM", u.DIANCode())

		s.DIANCode = "kilogramo"
		_, err = domain.RehydrateUnit(s)
		require.ErrorIs(t, err, domain.ErrInvalidUnitDIANCode)
	})
}
