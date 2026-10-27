package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// draft is a product of an import file in meters ("m"), with variants.
func draft(name, category string, variants ...app.VariantInput) app.ProductDraft {
	return app.ProductDraft{
		Name: name, Category: category, UnitCode: "M", TaxRate: dec("0.19"), Variants: variants,
	}
}

func TestImportProductDrafts(t *testing.T) {
	tests := []struct {
		business string
		draft    app.ProductDraft
	}{
		{"almacén de ropa", draft("Camiseta básica", "Camisetas",
			variantInput("cam-s-rojo", map[string]string{"talla": "S", "color": "Rojo"}),
			variantInput("cam-m-rojo", map[string]string{"talla": "M", "color": "Rojo"}))},
		{"ferretería", draft("Pintura vinilo blanca", "Pinturas",
			variantInput("pin-galon", map[string]string{"presentación": "Galón"}),
			variantInput("pin-cuarto", map[string]string{"presentación": "Cuarto"}))},
	}
	for _, tt := range tests {
		t.Run(tt.business, func(t *testing.T) {
			t.Run("crea un producto con sus variantes y la categoría que falta", func(t *testing.T) {
				e := newEnv(t)

				results, err := e.svc.ImportProductDrafts(t.Context(), e.actor, []app.ProductDraft{tt.draft}, app.UpdateExisting)
				require.NoError(t, err)

				require.Len(t, results, 1)
				require.NoError(t, results[0].Err)
				require.Equal(t, app.DraftCreated, results[0].Outcome)
				require.Len(t, results[0].Variants, 2)
				require.True(t, results[0].Variants[0].Created)
				require.Len(t, e.store.products, 1)
				require.Len(t, e.store.categories, 1)
				stored, err := e.svc.GetProduct(t.Context(), e.actor, results[0].ProductID)
				require.NoError(t, err)
				require.Len(t, stored.Variants(), 2)
				require.NotNil(t, stored.CategoryID())
				require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.products_created", "", ""))
				e.span(t, "catalog.ImportProductDrafts")
			})

			t.Run("revisar no guarda nada y dice qué pasaría", func(t *testing.T) {
				e := newEnv(t)

				results, err := e.svc.CheckProductDrafts(t.Context(), e.actor, []app.ProductDraft{tt.draft}, app.UpdateExisting)
				require.NoError(t, err)

				require.NoError(t, results[0].Err)
				require.Equal(t, app.DraftCreated, results[0].Outcome)
				require.Empty(t, e.store.products)
				require.Empty(t, e.store.categories)
				e.span(t, "catalog.CheckProductDrafts")
			})

			t.Run("una variante nueva de un producto existente se agrega", func(t *testing.T) {
				e := newEnv(t)
				first := tt.draft
				first.Variants = first.Variants[:1]
				_, err := e.svc.ImportProductDrafts(t.Context(), e.actor, []app.ProductDraft{first}, app.UpdateExisting)
				require.NoError(t, err)

				results, err := e.svc.ImportProductDrafts(t.Context(), e.actor, []app.ProductDraft{tt.draft}, app.UpdateExisting)
				require.NoError(t, err)

				require.NoError(t, results[0].Err)
				require.Equal(t, app.DraftUpdated, results[0].Outcome)
				require.False(t, results[0].Variants[0].Created)
				require.True(t, results[0].Variants[1].Created)
				require.Len(t, e.store.products, 1)
				require.Len(t, e.store.categories, 1)
			})
		})
	}
}

func TestImportProductDraftsExisting(t *testing.T) {
	existing := func(t *testing.T, e *env) *domain.Product {
		t.Helper()
		return e.createProduct(t, e.productInput(t)) // SKU TUB-PVC-12, price 10000
	}
	update := func(name, price string) app.ProductDraft {
		v := app.VariantInput{SKU: " tub-pvc-12 ", Cost: dec("8000"), Price: dec(price)}
		return draft(name, "", v)
	}

	t.Run("un SKU existente actualiza el producto en vez de duplicarlo", func(t *testing.T) {
		e := newEnv(t)
		p := existing(t, e)

		results, err := e.svc.ImportProductDrafts(t.Context(), e.actor,
			[]app.ProductDraft{update("Tubo PVC 1/2", "11500")}, app.UpdateExisting)
		require.NoError(t, err)

		require.NoError(t, results[0].Err)
		require.Equal(t, app.DraftUpdated, results[0].Outcome)
		require.Equal(t, p.ID(), results[0].ProductID)
		require.Len(t, e.store.products, 1)
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, "Tubo PVC 1/2", stored.Name())
		require.True(t, dec("11500").Equal(stored.DefaultVariant().Price()))
		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.price_changes", "", ""))
	})

	t.Run("con la política de omitir, un SKU existente no cambia", func(t *testing.T) {
		e := newEnv(t)
		p := existing(t, e)

		results, err := e.svc.ImportProductDrafts(t.Context(), e.actor,
			[]app.ProductDraft{update("Tubo PVC 1/2", "11500")}, app.SkipExisting)
		require.NoError(t, err)

		require.Equal(t, app.DraftSkipped, results[0].Outcome)
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, p.Name(), stored.Name())
	})

	t.Run("un SKU existente con otra unidad base y movimientos es un error del borrador", func(t *testing.T) {
		e := newEnv(t)
		p := existing(t, e)
		e.store.moved[p.DefaultVariant().ID()] = true
		e.addUnit(t, e.actor.TenantID, "kg", "Kilogramo")
		d := update("Tubo PVC 1/2", "10000")
		d.UnitCode = "kg"

		results, err := e.svc.CheckProductDrafts(t.Context(), e.actor, []app.ProductDraft{d}, app.UpdateExisting)
		require.NoError(t, err)
		require.ErrorIs(t, results[0].Err, domain.ErrBaseUnitLocked)
	})

	t.Run("variantes de dos productos distintos en un borrador es un error", func(t *testing.T) {
		e := newEnv(t)
		existing(t, e)
		other := e.productInput(t)
		other.SKU = "otro-1"
		e.createProduct(t, other)
		d := update("Tubo PVC 1/2", "10000")
		d.Variants = append(d.Variants, app.VariantInput{SKU: "otro-1", Cost: dec("1"), Price: dec("2")})

		results, err := e.svc.CheckProductDrafts(t.Context(), e.actor, []app.ProductDraft{d}, app.UpdateExisting)
		require.NoError(t, err)
		require.ErrorIs(t, results[0].Err, domain.ErrVariantsOfSeveralProducts)
	})
}

func TestCheckProductDraftsErrors(t *testing.T) {
	t.Run("un atributo obligatorio vacío es un error del borrador", func(t *testing.T) {
		tests := []struct {
			business, key, label string
		}{
			{"ferretería", "marca", "Marca"},
			{"droguería", "principio_activo", "Principio activo"},
		}
		for _, tt := range tests {
			t.Run(tt.business, func(t *testing.T) {
				e := newEnv(t)
				e.addAttribute(t, e.actor.TenantID, domain.AttributeParams{
					Key: tt.key, Label: tt.label, Type: domain.AttributeText, Required: true,
				})
				d := draft("Producto", "", app.VariantInput{SKU: "p-1", Cost: dec("1"), Price: dec("2")})

				results, err := e.svc.CheckProductDrafts(t.Context(), e.actor, []app.ProductDraft{d}, app.UpdateExisting)
				require.NoError(t, err)
				require.ErrorIs(t, results[0].Err, domain.ErrAttributeRequired)
			})
		}
	})

	t.Run("una unidad que no existe es un error del borrador", func(t *testing.T) {
		e := newEnv(t)
		d := draft("Producto", "", app.VariantInput{SKU: "p-1", Cost: dec("1"), Price: dec("2")})
		d.UnitCode = "bulto"

		results, err := e.svc.CheckProductDrafts(t.Context(), e.actor, []app.ProductDraft{d}, app.UpdateExisting)
		require.NoError(t, err)
		require.ErrorIs(t, results[0].Err, domain.ErrUnitNotFound)
	})

	t.Run("importar se detiene en el primer borrador con error", func(t *testing.T) {
		e := newEnv(t)
		ok := draft("Producto A", "", app.VariantInput{SKU: "a-1", Cost: dec("1"), Price: dec("2")})
		bad := draft("Producto B", "", app.VariantInput{SKU: "b-1", Cost: dec("1"), Price: dec("0")})
		never := draft("Producto C", "", app.VariantInput{SKU: "c-1", Cost: dec("1"), Price: dec("2")})

		results, err := e.svc.ImportProductDrafts(t.Context(), e.actor, []app.ProductDraft{ok, bad, never}, app.UpdateExisting)
		require.NoError(t, err)

		require.Len(t, results, 2)
		require.ErrorIs(t, results[1].Err, domain.ErrInvalidPrice)
		require.Len(t, e.store.products, 1)
	})
}
