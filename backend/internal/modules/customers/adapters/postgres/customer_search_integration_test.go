//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestCustomerSearch(t *testing.T) {
	f := newFixture(t)
	// A tenant with customers of a hardware store and of a clothing store.
	tenant := f.createTenant(t)
	builders := f.createSegment(t, tenant, "Maestro de obra").ID()
	wholesale := f.createSegment(t, tenant, "Mayorista").ID()
	inSegment := func(p domain.CustomerParams, segment uuid.UUID) domain.CustomerParams {
		p.SegmentID = &segment
		return p
	}
	f.createCustomer(t, tenant, inSegment(person("1020304050", "José Gómez Ramírez"), builders))
	f.createCustomer(t, tenant, company("900123456-8", "Confecciones del Valle S.A.S.", &wholesale))
	f.createCustomer(t, tenant, person("10203040", "Marta Gómez"))
	inactive := person("79845123", "Gómez Inactivo")
	inactive.Active = false
	f.createCustomer(t, tenant, inactive)

	search := func(t *testing.T, flt domain.CustomerFilter) ([]string, int) {
		t.Helper()
		flt.TenantID = tenant
		if flt.Status == "" {
			flt.Status = domain.StatusActive
		}
		if flt.Limit == 0 {
			flt.Limit = 20
		}
		items, total, err := f.customers.Search(t.Context(), flt)
		require.NoError(t, err)
		return names(items), total
	}

	t.Run("sin texto lista los activos por nombre", func(t *testing.T) {
		got, total := search(t, domain.CustomerFilter{})
		require.Equal(t, []string{"Confecciones del Valle S.A.S.", "José Gómez Ramírez", "Marta Gómez"}, got)
		require.Equal(t, 3, total)
	})

	t.Run("busca por palabras del nombre sin tildes ni mayúsculas", func(t *testing.T) {
		got, _ := search(t, domain.CustomerFilter{Text: "gomez JOSE"})
		require.Equal(t, []string{"José Gómez Ramírez"}, got)
		got, _ = search(t, domain.CustomerFilter{Text: "la moda"})
		require.Equal(t, []string{"Confecciones del Valle S.A.S."}, got, "busca también el nombre comercial")
	})

	t.Run("busca por documento y pone primero el exacto", func(t *testing.T) {
		got, total := search(t, domain.CustomerFilter{Text: "10.203.040"})
		require.Equal(t, []string{"Marta Gómez", "José Gómez Ramírez"}, got)
		require.Equal(t, 2, total)
		got, _ = search(t, domain.CustomerFilter{Text: "900123456-8"})
		require.Equal(t, []string{"Confecciones del Valle S.A.S."}, got)
	})

	t.Run("filtra por segmento y por estado", func(t *testing.T) {
		got, _ := search(t, domain.CustomerFilter{SegmentID: &builders})
		require.Equal(t, []string{"José Gómez Ramírez"}, got)
		got, _ = search(t, domain.CustomerFilter{Status: domain.StatusInactive})
		require.Equal(t, []string{"Gómez Inactivo"}, got)
		got, total := search(t, domain.CustomerFilter{Status: domain.StatusAll, Text: "gomez", Limit: 2, Offset: 1})
		require.Len(t, got, 2)
		require.Equal(t, 3, total)
	})
}
