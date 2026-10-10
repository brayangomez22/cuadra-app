package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestParseEmail(t *testing.T) {
	t.Run("acepta email válido y lo normaliza a minúsculas sin espacios", func(t *testing.T) {
		tests := []struct {
			in   string
			want string
		}{
			{in: "dueno@ferreteria.co", want: "dueno@ferreteria.co"},
			{in: "  Juan.Perez@Ferreteria-El-Tornillo.COM.co ", want: "juan.perez@ferreteria-el-tornillo.com.co"},
			{in: "caja+1@gmail.com", want: "caja+1@gmail.com"},
		}
		for _, tt := range tests {
			got, err := domain.ParseEmail(tt.in)
			require.NoError(t, err, tt.in)
			require.Equal(t, tt.want, got.String())
			require.False(t, got.IsZero())
		}
	})

	t.Run("rechaza email inválido", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
		}{
			{name: "vacío", in: ""},
			{name: "solo espacios", in: "   "},
			{name: "sin arroba", in: "dueno.ferreteria.co"},
			{name: "sin parte local", in: "@ferreteria.co"},
			{name: "sin dominio", in: "dueno@"},
			{name: "dominio sin TLD", in: "dueno@ferreteria"},
			{name: "con nombre visible", in: "Juan <juan@ferreteria.co>"},
			{name: "con espacios internos", in: "juan perez@ferreteria.co"},
			{name: "dos arrobas", in: "juan@@ferreteria.co"},
			{name: "más de 254 caracteres", in: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 186) + ".com"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.ParseEmail(tt.in)
				require.ErrorIs(t, err, domain.ErrInvalidEmail)
			})
		}
	})
}
