//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestSupplierRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee un proveedor con su contacto", func(t *testing.T) {
		tn := f.createTenant(t)
		s := f.addSupplier(t, tn, "Distribuidora de Tornillos del Valle", "900.123.456-8")
		got, err := f.suppliers.GetByID(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, s.Snapshot(), got.Snapshot())
	})

	t.Run("no hay dos proveedores con el mismo NIT en el tenant", func(t *testing.T) {
		tn := f.createTenant(t)
		f.addSupplier(t, tn, "Confecciones Andinas", "901234567-7")
		dup, err := domain.NewSupplier(tn.id, domain.SupplierParams{Name: "Otra", NIT: "9012345677", Active: true}, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.suppliers.Create(t.Context(), dup), domain.ErrNITTaken)
	})

	t.Run("busca por palabras del nombre sin tildes o por el comienzo del NIT", func(t *testing.T) {
		tn := f.createTenant(t)
		f.addSupplier(t, tn, "Droguería Mayorista del Norte", "900123456-8")
		f.addSupplier(t, tn, "Calzado Andino", "901234567-7")
		for _, tc := range []struct {
			text string
			want int
		}{{"drogueria norte", 1}, {"9012", 1}, {"90", 2}, {"", 2}, {"papelería", 0}} {
			items, total, err := f.suppliers.Search(t.Context(), domain.SupplierFilter{
				TenantID: tn.id, Text: tc.text, Status: domain.SuppliersAll, Limit: 20,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, total, tc.text)
			require.Len(t, items, tc.want, tc.text)
		}
	})

	t.Run("filtra por estado y edita", func(t *testing.T) {
		tn := f.createTenant(t)
		s := f.addSupplier(t, tn, "Calzado Andino", "901234567-7")
		require.NoError(t, s.Update(domain.SupplierParams{Name: "Calzado Andino SAS", NIT: "901234567-7", Active: false}))
		require.NoError(t, f.suppliers.Update(t.Context(), s))
		_, total, err := f.suppliers.Search(t.Context(), domain.SupplierFilter{TenantID: tn.id, Status: domain.SuppliersActive, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, total)
		items, _, err := f.suppliers.Search(t.Context(), domain.SupplierFilter{TenantID: tn.id, Status: domain.SuppliersInactive, Limit: 20})
		require.NoError(t, err)
		require.Equal(t, "Calzado Andino SAS", items[0].Name())
	})

	t.Run("aislamiento: el tenant B no ve ni modifica los proveedores del A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := f.addSupplier(t, a, "Distribuidora del Valle", "900123456-8")

		_, err := f.suppliers.GetByID(t.Context(), b.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSupplierNotFound)
		_, total, err := f.suppliers.Search(t.Context(), domain.SupplierFilter{TenantID: b.id, Status: domain.SuppliersAll, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, total)

		snap := s.Snapshot()
		snap.TenantID = b.id
		snap.Name = "Robado"
		forged, err := domain.RehydrateSupplier(snap)
		require.NoError(t, err)
		require.ErrorIs(t, f.suppliers.Update(t.Context(), forged), domain.ErrSupplierNotFound)
		got, err := f.suppliers.GetByID(t.Context(), a.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, "Distribuidora del Valle", got.Name())

		// El mismo NIT sí puede ser proveedor de otro tenant.
		f.addSupplier(t, b, "Distribuidora del Valle", "900123456-8")
	})
}
