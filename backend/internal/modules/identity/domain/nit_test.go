package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// Public NITs of well-known Colombian companies, taken from their RUT.
func TestParseNIT(t *testing.T) {
	t.Run("acepta NIT real con dígito de verificación correcto", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
			want string
		}{
			{name: "Bancolombia", in: "890903938-8", want: "890903938-8"},
			{name: "Ecopetrol", in: "899999068-1", want: "899999068-1"},
			{name: "DIAN", in: "800197268-4", want: "800197268-4"},
			{name: "Almacenes Éxito", in: "890900608-9", want: "890900608-9"},
			{name: "Davivienda", in: "860034313-7", want: "860034313-7"},
			{name: "con puntos", in: "890.903.938-8", want: "890903938-8"},
			{name: "sin guion", in: "8909039388", want: "890903938-8"},
			{name: "con espacios", in: " 890 903 938 - 8 ", want: "890903938-8"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := domain.ParseNIT(tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.want, got.String())
				require.False(t, got.IsZero())
			})
		}
	})

	t.Run("rechaza NIT con dígito de verificación incorrecto", func(t *testing.T) {
		for _, in := range []string{"890903938-7", "899999068-0", "800197268-5", "8909006081"} {
			_, err := domain.ParseNIT(in)
			require.ErrorIs(t, err, domain.ErrInvalidNITCheckDigit, in)
		}
	})

	t.Run("rechaza NIT con caracteres no numéricos o longitud inválida", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
		}{
			{name: "vacío", in: ""},
			{name: "con letras", in: "89090A938-8"},
			{name: "sin dígito de verificación", in: "890903938-"},
			{name: "dígito de verificación de dos cifras", in: "890903938-88"},
			{name: "dos guiones", in: "890-903938-8"},
			{name: "número demasiado corto", in: "12-3"},
			{name: "número de más de 15 dígitos", in: "1234567890123456-1"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.ParseNIT(tt.in)
				require.ErrorIs(t, err, domain.ErrInvalidNIT)
			})
		}
	})
}
