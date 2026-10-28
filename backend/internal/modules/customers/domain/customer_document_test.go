package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestParseDocument(t *testing.T) {
	t.Run("acepta cada tipo de documento y lo deja en su forma canónica", func(t *testing.T) {
		tests := []struct {
			name string
			typ  domain.DocumentType
			in   string
			want string
		}{
			{name: "NIT con guion", typ: domain.DocumentNIT, in: "900123456-8", want: "900123456-8"},
			{name: "NIT con puntos", typ: domain.DocumentNIT, in: "900.123.456-8", want: "900123456-8"},
			{name: "NIT sin guion", typ: domain.DocumentNIT, in: "9001234568", want: "900123456-8"},
			{name: "cédula con puntos", typ: domain.DocumentCC, in: "1.020.304.050", want: "1020304050"},
			{name: "cédula con espacios", typ: domain.DocumentCC, in: " 79 845 123 ", want: "79845123"},
			{name: "cédula de extranjería", typ: domain.DocumentCE, in: "456789", want: "456789"},
			{name: "pasaporte en minúsculas", typ: domain.DocumentPassport, in: "ab123456", want: "AB123456"},
			{name: "pasaporte con guion", typ: domain.DocumentPassport, in: "PE-1234567", want: "PE1234567"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				doc, err := domain.ParseDocument(tt.typ, tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.typ, doc.Type())
				require.Equal(t, tt.want, doc.Number())
			})
		}
	})

	t.Run("rechaza NIT con dígito de verificación inválido", func(t *testing.T) {
		_, err := domain.ParseDocument(domain.DocumentNIT, "900123456-1")
		require.ErrorIs(t, err, domain.ErrInvalidNITCheckDigit)
	})

	t.Run("rechaza documentos que no corresponden a su tipo", func(t *testing.T) {
		tests := []struct {
			name string
			typ  domain.DocumentType
			in   string
		}{
			{name: "NIT con letras", typ: domain.DocumentNIT, in: "90012A456-8"},
			{name: "cédula con letras", typ: domain.DocumentCC, in: "1020A04050"},
			{name: "cédula de más de 10 dígitos", typ: domain.DocumentCC, in: "12345678901"},
			{name: "cédula muy corta", typ: domain.DocumentCC, in: "12"},
			{name: "cédula de extranjería con letras", typ: domain.DocumentCE, in: "CE4567"},
			{name: "pasaporte con símbolos", typ: domain.DocumentPassport, in: "AB#12345"},
			{name: "pasaporte de más de 20 caracteres", typ: domain.DocumentPassport, in: "A12345678901234567890"},
			{name: "vacío", typ: domain.DocumentCC, in: "  "},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.ParseDocument(tt.typ, tt.in)
				require.ErrorIs(t, err, domain.ErrInvalidDocument)
			})
		}
	})

	t.Run("rechaza un tipo de documento desconocido", func(t *testing.T) {
		_, err := domain.ParseDocument("ti", "1020304050")
		require.ErrorIs(t, err, domain.ErrInvalidDocumentType)
	})

	t.Run("muestra el documento con su tipo", func(t *testing.T) {
		doc, err := domain.ParseDocument(domain.DocumentNIT, "900123456-8")
		require.NoError(t, err)
		require.Equal(t, "NIT 900123456-8", doc.String())
	})
}
