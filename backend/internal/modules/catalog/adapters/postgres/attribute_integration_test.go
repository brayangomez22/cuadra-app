//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func (f fixture) createAttribute(t *testing.T, tenantID uuid.UUID, p domain.AttributeParams) *domain.AttributeDefinition {
	t.Helper()
	d, err := domain.NewAttributeDefinition(tenantID, p, now)
	require.NoError(t, err)
	require.NoError(t, f.attributes.Create(t.Context(), d))
	return d
}

var (
	materialParams = domain.AttributeParams{
		Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Algodón", "Cobre"},
	}
	brandParams = domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true}
)

// attributesOf validates values against defs, as the use cases do.
func attributesOf(t *testing.T, defs []*domain.AttributeDefinition, values map[string]domain.AttributeValue) domain.Attributes {
	t.Helper()
	attrs, err := domain.ResolveAttributes(defs, values)
	require.NoError(t, err)
	return attrs
}

func TestAttributeRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, lee, edita y lista las definiciones del tenant por etiqueta", func(t *testing.T) {
		tenant := f.createTenant(t)
		material := f.createAttribute(t, tenant, materialParams)
		f.createAttribute(t, tenant, brandParams)

		got, err := f.attributes.GetByID(t.Context(), tenant, material.ID())
		require.NoError(t, err)
		require.Equal(t, tenant, got.TenantID())
		require.Equal(t, "material", got.Key())
		require.Equal(t, domain.AttributeList, got.Type())
		require.Equal(t, []string{"PVC", "Algodón", "Cobre"}, got.Options())
		require.Equal(t, now, got.CreatedAt())

		_, err = material.Update("Material principal", []string{"PVC", "Algodón", "Cobre", "Lino"}, true)
		require.NoError(t, err)
		require.NoError(t, f.attributes.Update(t.Context(), material, nil))

		list, err := f.attributes.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Marca", list[0].Label())
		require.Equal(t, "Material principal", list[1].Label())
		require.Equal(t, []string{"PVC", "Algodón", "Cobre", "Lino"}, list[1].Options())
		require.True(t, list[1].Required())
	})

	t.Run("una definición inexistente da ErrAttributeNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		missing := uuid.Must(uuid.NewV7())

		_, err := f.attributes.GetByID(t.Context(), tenant, missing)
		require.ErrorIs(t, err, domain.ErrAttributeNotFound)
		require.ErrorIs(t, f.attributes.Delete(t.Context(), tenant, missing), domain.ErrAttributeNotFound)
		ghost, err := domain.NewAttributeDefinition(tenant, brandParams, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.attributes.Update(t.Context(), ghost, nil), domain.ErrAttributeNotFound)
	})

	t.Run("rechaza dos definiciones con la misma clave en el tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createAttribute(t, tenant, brandParams)

		dup, err := domain.NewAttributeDefinition(tenant, domain.AttributeParams{Key: "marca", Label: "Fabricante", Type: domain.AttributeText}, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.attributes.Create(t.Context(), dup), domain.ErrAttributeKeyTaken)
	})

	t.Run("los atributos del producto se guardan y se leen", func(t *testing.T) {
		tenant := f.createTenant(t)
		defs := []*domain.AttributeDefinition{
			f.createAttribute(t, tenant, brandParams),
			f.createAttribute(t, tenant, materialParams),
			f.createAttribute(t, tenant, domain.AttributeParams{Key: "hojas", Label: "Hojas", Type: domain.AttributeNumber}),
			f.createAttribute(t, tenant, domain.AttributeParams{Key: "importado", Label: "Importado", Type: domain.AttributeBoolean}),
		}
		want := map[string]domain.AttributeValue{
			"marca": domain.TextValue("Acme"), "material": domain.TextValue("PVC"),
			"hojas": domain.TextValue("2.5"), "importado": domain.BoolValue(false),
		}
		p := f.createProduct(t, tenant, "TUB-1", "Tubo", withAttributes(attributesOf(t, defs, want)))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Equal(t, want, got.Attributes().Values())

		changed := map[string]domain.AttributeValue{"marca": domain.TextValue("Textiles"), "importado": domain.BoolValue(true)}
		require.NoError(t, got.UpdateDetails(domain.ProductDetails{
			Name: got.Name(), BaseUnit: got.BaseUnit(), TaxRate: got.TaxRate(), Attributes: attributesOf(t, defs, changed),
		}))
		require.NoError(t, f.products.Update(t.Context(), got))

		page, err := f.products.Search(t.Context(), tenant, domain.ProductFilter{Limit: 10})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Equal(t, changed, page.Items[0].Attributes().Values())
	})

	t.Run("un producto sin atributos los lee vacíos", func(t *testing.T) {
		tenant := f.createTenant(t)
		p := f.createProduct(t, tenant, "CAM-1", "Camiseta")

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Empty(t, got.Attributes().Values())
	})

	t.Run("no se puede borrar un atributo que usa un producto", func(t *testing.T) {
		tenant := f.createTenant(t)
		brand := f.createAttribute(t, tenant, brandParams)
		unused := f.createAttribute(t, tenant, materialParams)
		f.createProduct(t, tenant, "TUB-1", "Tubo", withAttributes(attributesOf(t,
			[]*domain.AttributeDefinition{brand}, map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")})))

		require.ErrorIs(t, f.attributes.Delete(t.Context(), tenant, brand.ID()), domain.ErrAttributeInUse)

		require.NoError(t, f.attributes.Delete(t.Context(), tenant, unused.ID()))
		_, err := f.attributes.GetByID(t.Context(), tenant, unused.ID())
		require.ErrorIs(t, err, domain.ErrAttributeNotFound)
	})

	t.Run("no se puede quitar una opción en uso", func(t *testing.T) {
		tenant := f.createTenant(t)
		material := f.createAttribute(t, tenant, materialParams)
		f.createProduct(t, tenant, "CAM-1", "Camiseta", withAttributes(attributesOf(t,
			[]*domain.AttributeDefinition{material}, map[string]domain.AttributeValue{"material": domain.TextValue("Algodón")})))

		removed, err := material.Update("Material", []string{"PVC", "Cobre"}, false)
		require.NoError(t, err)
		require.ErrorIs(t, f.attributes.Update(t.Context(), material, removed), domain.ErrAttributeOptionInUse)

		stored, err := f.attributes.GetByID(t.Context(), tenant, material.ID())
		require.NoError(t, err)
		require.Equal(t, []string{"PVC", "Algodón", "Cobre"}, stored.Options(), "the update rolled back")

		removed, err = stored.Update("Material", []string{"PVC", "Algodón"}, false)
		require.NoError(t, err)
		require.NoError(t, f.attributes.Update(t.Context(), stored, removed), "quitar una opción sin uso sí se puede")
	})
}

func TestAttributeTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	brandA := f.createAttribute(t, tenantA, brandParams)

	t.Run("dos tenants pueden usar la misma clave", func(t *testing.T) {
		f.createAttribute(t, tenantB, brandParams)
	})

	t.Run("el tenant B no ve, no edita ni borra las definiciones del tenant A", func(t *testing.T) {
		_, err := f.attributes.GetByID(t.Context(), tenantB, brandA.ID())
		require.ErrorIs(t, err, domain.ErrAttributeNotFound)

		list, err := f.attributes.List(t.Context(), tenantB)
		require.NoError(t, err)
		for _, d := range list {
			require.Equal(t, tenantB, d.TenantID())
		}

		require.ErrorIs(t, f.attributes.Delete(t.Context(), tenantB, brandA.ID()), domain.ErrAttributeNotFound)
		stored, err := f.attributes.GetByID(t.Context(), tenantA, brandA.ID())
		require.NoError(t, err)
		require.Equal(t, "Marca", stored.Label())
	})

	t.Run("los productos del tenant B no cuentan como uso de las definiciones del tenant A", func(t *testing.T) {
		brandB, err := f.attributes.List(t.Context(), tenantB)
		require.NoError(t, err)
		f.createProduct(t, tenantB, "TUB-1", "Tubo", withAttributes(attributesOf(t,
			brandB, map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")})))

		require.NoError(t, f.attributes.Delete(t.Context(), tenantA, brandA.ID()))
	})
}
