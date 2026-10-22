//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// setUnit gives p's default variant an alternate unit and stores p.
func (f fixture) setUnit(t *testing.T, p *domain.Product, unit domain.UnitRef, factor, price, barcode string) error {
	t.Helper()
	require.NoError(t, p.SetVariantUnit(p.DefaultVariant().ID(), domain.VariantUnitParams{
		Unit: unit, Factor: dec(factor), Price: dec(price), Barcode: barcode,
	}))
	return f.products.Update(t.Context(), p)
}

func TestVariantUnits(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name, code, unitName string
		factor, price        string
	}{
		{"ferretería: caja de 12", "caja", "Caja", "12", "110000"},
		{"almacén de ropa: paquete de 3", "paquete", "Paquete", "3", "70000.5"},
		{"tienda: gramo de un producto por kilo", "g", "Gramo", "0.001", "12.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("guarda y lee las unidades alternativas de una variante", func(t *testing.T) {
				tenant := f.createTenant(t)
				unit := f.createUnit(t, tenant, tt.code, tt.unitName)
				p := f.createProduct(t, tenant, "SKU-"+tt.code, "Producto")
				require.NoError(t, f.setUnit(t, p, unit.Ref(), tt.factor, tt.price, "7708"+tt.code))

				got, err := f.products.GetByID(t.Context(), tenant, p.ID())
				require.NoError(t, err)
				units := got.DefaultVariant().Units()
				require.Len(t, units, 1)
				require.Equal(t, unit.Ref(), units[0].Unit())
				require.True(t, dec(tt.factor).Equal(units[0].Factor()))
				require.True(t, dec(tt.price).Equal(units[0].Price()))
				require.Equal(t, "7708"+tt.code, units[0].Barcode())

				page, err := f.products.Search(t.Context(), tenant, domain.ProductFilter{Query: "7708" + tt.code, Limit: 20})
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				require.Len(t, page.Items[0].DefaultVariant().Units(), 1)
				require.Equal(t, 1, page.Total)
			})
		})
	}

	t.Run("quitar una unidad la borra de la base", func(t *testing.T) {
		tenant := f.createTenant(t)
		box := f.createUnit(t, tenant, "caja", "Caja")
		p := f.createProduct(t, tenant, "TOR-1", "Tornillo")
		require.NoError(t, f.setUnit(t, p, box.Ref(), "12", "5500", ""))

		require.NoError(t, p.RemoveVariantUnit(p.DefaultVariant().ID(), box.ID()))
		require.NoError(t, f.products.Update(t.Context(), p))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Empty(t, got.DefaultVariant().Units())
	})

	t.Run("un código de barras no se repite entre variantes y unidades del tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		box := f.createUnit(t, tenant, "caja", "Caja")
		withCode := f.newProduct(t, tenant, "CON-1", "Con código")
		require.NoError(t, withCode.UpdateVariant(withCode.DefaultVariant().ID(), domain.VariantDetails{
			SKU: "CON-1", Barcode: "7701010101010", Cost: dec("1"),
		}))
		require.NoError(t, f.products.Create(t.Context(), withCode))
		p := f.createProduct(t, tenant, "OTRO-1", "Otro")

		require.ErrorIs(t, f.setUnit(t, p, box.Ref(), "12", "5500", "7701010101010"), domain.ErrBarcodeTaken)

		require.NoError(t, f.setUnit(t, p, box.Ref(), "12", "5500", "7702020202020"))
		third := f.newProduct(t, tenant, "TERCERO-1", "Tercero")
		require.NoError(t, third.UpdateVariant(third.DefaultVariant().ID(), domain.VariantDetails{
			SKU: "TERCERO-1", Barcode: "7702020202020", Cost: dec("1"),
		}))
		require.ErrorIs(t, f.products.Create(t.Context(), third), domain.ErrBarcodeTaken)

		fourth := f.createProduct(t, tenant, "CUARTO-1", "Cuarto")
		require.ErrorIs(t, f.setUnit(t, fourth, box.Ref(), "6", "3000", "7702020202020"), domain.ErrBarcodeTaken)
	})

	t.Run("una unidad usada como alternativa no se puede borrar", func(t *testing.T) {
		tenant := f.createTenant(t)
		box := f.createUnit(t, tenant, "caja", "Caja")
		p := f.createProduct(t, tenant, "TOR-2", "Tornillo")
		require.NoError(t, f.setUnit(t, p, box.Ref(), "12", "5500", ""))

		require.ErrorIs(t, f.units.Delete(t.Context(), tenant, box.ID()), domain.ErrUnitInUse)
	})
}

func TestVariantUnitTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	boxA := f.createUnit(t, tenantA, "caja", "Caja")
	productA := f.createProduct(t, tenantA, "TOR-1", "Tornillo")
	require.NoError(t, f.setUnit(t, productA, boxA.Ref(), "12", "5500", "7704040404040"))

	t.Run("el tenant B no encuentra el producto del tenant A por el código de su unidad", func(t *testing.T) {
		page, err := f.products.Search(t.Context(), tenantB, domain.ProductFilter{Query: "7704040404040", Limit: 20})
		require.NoError(t, err)
		require.Empty(t, page.Items)
	})

	t.Run("el tenant B puede usar el mismo código de barras en sus unidades", func(t *testing.T) {
		boxB := f.createUnit(t, tenantB, "caja", "Caja")
		p := f.createProduct(t, tenantB, "TOR-1", "Tornillo")
		require.NoError(t, f.setUnit(t, p, boxB.Ref(), "12", "5500", "7704040404040"))
	})

	t.Run("el tenant B no cambia las unidades de una variante del tenant A", func(t *testing.T) {
		boxB := f.createUnit(t, tenantB, "caja-b", "Caja B")
		forged, err := domain.RehydrateProduct(domain.ProductSnapshot{
			ID: productA.ID(), TenantID: tenantB, Name: "Tornillo", BaseUnit: f.piece[tenantB],
			TaxRate: productA.TaxRate(), Active: true, CreatedAt: now,
			Variants: []domain.VariantSnapshot{{
				ID: productA.DefaultVariant().ID(), SKU: "HACK-1", Cost: dec("1"), Price: dec("1"),
				IsDefault: true, Active: true, CreatedAt: now,
				Units: []domain.VariantUnitParams{{Unit: boxB.Ref(), Factor: dec("1000"), Price: dec("1")}},
			}},
		})
		require.NoError(t, err)

		require.ErrorIs(t, f.products.Update(t.Context(), forged), domain.ErrProductNotFound)

		got, err := f.products.GetByID(t.Context(), tenantA, productA.ID())
		require.NoError(t, err)
		units := got.DefaultVariant().Units()
		require.Len(t, units, 1)
		require.Equal(t, boxA.ID(), units[0].Unit().ID)
		require.True(t, dec("12").Equal(units[0].Factor()))
	})

	t.Run("el tenant B no lee unidades de una variante del tenant A", func(t *testing.T) {
		_, err := f.products.GetByID(t.Context(), tenantB, productA.ID())
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}
