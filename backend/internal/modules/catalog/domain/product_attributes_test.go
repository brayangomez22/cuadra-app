package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestProductAttributes(t *testing.T) {
	t.Run("un producto nuevo guarda sus atributos", func(t *testing.T) {
		attrs, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Acme"), "material": text("PVC"),
		})
		require.NoError(t, err)
		params := validProductParams(t)
		params.Attributes = attrs

		p, err := domain.NewProduct(params, testNow)
		require.NoError(t, err)

		require.Equal(t, map[string]domain.AttributeValue{"marca": text("Acme"), "material": text("PVC")}, p.Attributes().Values())
	})

	t.Run("un producto sin atributos los tiene vacíos", func(t *testing.T) {
		p, err := domain.NewProduct(validProductParams(t), testNow)
		require.NoError(t, err)
		require.Empty(t, p.Attributes().Values())
	})

	t.Run("editar los datos reemplaza los atributos", func(t *testing.T) {
		p := mustProduct(t, "28000", "32500")
		attrs, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Textiles"), "material": text("Algodón"),
		})
		require.NoError(t, err)

		require.NoError(t, p.UpdateDetails(domain.ProductDetails{
			Name: "Camiseta básica", BaseUnit: piece, TaxRate: mustTaxRate(t, "0.19"), Attributes: attrs,
		}))

		require.Equal(t, attrs.Values(), p.Attributes().Values())
	})

	t.Run("rehidratar conserva los atributos", func(t *testing.T) {
		attrs, err := domain.RehydrateAttributes(map[string]domain.AttributeValue{"importado": domain.BoolValue(true)})
		require.NoError(t, err)
		p := mustProduct(t, "28000", "32500")
		s := domain.ProductSnapshot{
			ID: p.ID(), TenantID: p.TenantID(), Name: p.Name(), BaseUnit: p.BaseUnit(), TaxRate: p.TaxRate(),
			Active: true, CreatedAt: p.CreatedAt(), Attributes: attrs,
			Variants: []domain.VariantSnapshot{{
				ID: p.DefaultVariant().ID(), SKU: "SKU-1", Price: dec("1000"), IsDefault: true, Active: true,
			}},
		}

		got, err := domain.RehydrateProduct(s)
		require.NoError(t, err)
		require.Equal(t, attrs.Values(), got.Attributes().Values())
	})
}
