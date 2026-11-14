package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestDocumentTypesOfTheDIAN(t *testing.T) {
	t.Run("acepta tarjeta de identidad, PPT, NIT de otro país y documento extranjero", func(t *testing.T) {
		tests := []struct {
			name string
			typ  domain.DocumentType
			in   string
			want string
		}{
			{name: "tarjeta de identidad con puntos", typ: domain.DocumentTI, in: "1.098.765.432", want: "1098765432"},
			{name: "PPT", typ: domain.DocumentPPT, in: "5 432 109", want: "5432109"},
			{name: "NIT de otro país con letras", typ: domain.DocumentForeignNIT, in: "b-12345678", want: "B12345678"},
			{name: "documento extranjero", typ: domain.DocumentForeignID, in: "dni 40123456", want: "DNI40123456"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				doc, err := domain.ParseDocument(tt.typ, tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.want, doc.Number())
			})
		}
	})

	t.Run("rechaza documentos nuevos que no corresponden a su tipo", func(t *testing.T) {
		for _, tt := range []struct {
			typ domain.DocumentType
			in  string
		}{
			{domain.DocumentTI, "10987A5432"},
			{domain.DocumentTI, "123456789012"},
			{domain.DocumentPPT, "PPT123"},
			{domain.DocumentForeignNIT, "B#123"},
			{domain.DocumentForeignID, "A12345678901234567890"},
		} {
			_, err := domain.ParseDocument(tt.typ, tt.in)
			require.ErrorIs(t, err, domain.ErrInvalidDocument, "%s %q", tt.typ, tt.in)
		}
	})

	t.Run("cada tipo de documento tiene su código DIAN", func(t *testing.T) {
		want := map[domain.DocumentType]string{
			domain.DocumentTI: "12", domain.DocumentCC: "13", domain.DocumentCE: "22", domain.DocumentNIT: "31",
			domain.DocumentPassport: "41", domain.DocumentForeignID: "42", domain.DocumentPPT: "48",
			domain.DocumentForeignNIT: "50",
		}
		require.ElementsMatch(t, domain.DocumentTypes(), keys(want))
		for typ, code := range want {
			require.Equal(t, code, typ.DIANCode(), typ)
		}
	})

	t.Run("una persona jurídica se identifica con NIT o NIT de otro país", func(t *testing.T) {
		p := company()
		p.DocumentType, p.DocumentNumber = domain.DocumentForeignNIT, "B12345678"
		_, err := domain.NewCustomer(tenantID, p, now)
		require.NoError(t, err)

		p.DocumentType, p.DocumentNumber = domain.DocumentForeignID, "X1234567"
		_, err = domain.NewCustomer(tenantID, p, now)
		require.ErrorIs(t, err, domain.ErrLegalPersonRequiresNIT)
	})
}

func TestCustomerMunicipality(t *testing.T) {
	t.Run("guarda el municipio DANE del cliente (ferretería en Medellín, almacén de ropa en Bogotá)", func(t *testing.T) {
		for _, code := range []string{"05001", "11001"} {
			p := person()
			p.Contact.MunicipalityCode = " " + code + " "
			c, err := domain.NewCustomer(tenantID, p, now)
			require.NoError(t, err)
			require.Equal(t, code, c.Contact().MunicipalityCode)
		}
	})

	t.Run("un cliente con ciudad en texto libre y sin municipio sigue funcionando", func(t *testing.T) {
		p := company()
		p.Contact.City, p.Contact.MunicipalityCode = "Medellín", ""
		c, err := domain.NewCustomer(tenantID, p, now)
		require.NoError(t, err)
		require.Equal(t, "Medellín", c.Contact().City)
		require.Empty(t, c.Contact().MunicipalityCode)
	})

	t.Run("rechaza un código de municipio que no tiene 5 dígitos", func(t *testing.T) {
		for _, code := range []string{"5001", "050011", "05A01"} {
			p := person()
			p.Contact.MunicipalityCode = code
			_, err := domain.NewCustomer(tenantID, p, now)
			require.ErrorIs(t, err, domain.ErrInvalidMunicipality, code)
		}
	})
}

func keys(m map[domain.DocumentType]string) []domain.DocumentType {
	out := make([]domain.DocumentType, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
