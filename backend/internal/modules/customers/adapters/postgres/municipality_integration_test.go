//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestMunicipalities(t *testing.T) {
	f := newFixture(t)

	t.Run("lee los municipios del DANE por departamento y nombre", func(t *testing.T) {
		list, err := postgres.NewMunicipalityCatalog(f.db).List(t.Context())
		require.NoError(t, err)

		require.Len(t, list, 1122)
		require.Equal(t, "Amazonas", list[0].DepartmentName)
		require.Contains(t, list, domain.Municipality{
			Code: "05001", Name: "Medellín", DepartmentCode: "05", DepartmentName: "Antioquia",
		})
	})

	t.Run("guarda y lee el municipio del cliente (ferretería en Medellín, almacén de ropa en Bogotá)", func(t *testing.T) {
		tenant := f.createTenant(t)
		for _, code := range []string{"05001", "11001"} {
			p := person("10203040"+code[:2], "Cliente "+code)
			p.Contact.MunicipalityCode = code
			c := f.createCustomer(t, tenant, p)

			got, err := f.customers.GetByID(t.Context(), tenant, c.ID())
			require.NoError(t, err)
			require.Equal(t, code, got.Contact().MunicipalityCode)
		}
	})

	t.Run("rechaza un municipio que no existe al crear y al editar", func(t *testing.T) {
		tenant := f.createTenant(t)
		p := person("1020304050", "Carlos Pérez")
		p.Contact.MunicipalityCode = "99999"
		c, err := domain.NewCustomer(tenant, p, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.customers.Create(t.Context(), c), domain.ErrInvalidMunicipality)

		ok := f.createCustomer(t, tenant, person("79845123", "Ana Gómez"))
		q := person("79845123", "Ana Gómez")
		q.Contact.MunicipalityCode = "99999"
		require.NoError(t, ok.Update(q))
		require.ErrorIs(t, f.customers.Update(t.Context(), ok), domain.ErrInvalidMunicipality)
	})

	t.Run("guarda los tipos de documento nuevos de la DIAN", func(t *testing.T) {
		tenant := f.createTenant(t)
		p := company("900123456-8", "Textiles Ibéricos S.L.", nil)
		p.DocumentType, p.DocumentNumber = domain.DocumentForeignNIT, "B12345678"
		c := f.createCustomer(t, tenant, p)

		got, err := f.customers.GetByID(t.Context(), tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.DocumentForeignNIT, got.Document().Type())
	})

	t.Run("aislamiento: el municipio del cliente del tenant A no se ve desde el B", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		p := person("1020304050", "Carlos Pérez")
		p.Contact.MunicipalityCode = "05001"
		c := f.createCustomer(t, a, p)

		_, err := f.customers.GetByID(t.Context(), b, c.ID())
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}
