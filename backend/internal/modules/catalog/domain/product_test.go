package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func mustTaxRate(t *testing.T, s string) domain.TaxRate {
	t.Helper()
	rate, err := domain.ParseTaxRate(dec(s))
	require.NoError(t, err)
	return rate
}

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

func validProductParams(t *testing.T) domain.NewProductParams {
	t.Helper()
	return domain.NewProductParams{
		TenantID:    uuid.Must(uuid.NewV7()),
		Name:        "  Cemento gris 50kg  ",
		Description: "Cemento de uso general",
		BaseUnit:    domain.UnitBulto,
		TaxRate:     mustTaxRate(t, "0.19"),
		Variant: domain.VariantParams{
			SKU:     "  cem-gris-50  ",
			Barcode: "7701234567890",
			Cost:    dec("28000"),
			Price:   dec("32500.50"),
		},
	}
}

func mustProduct(t *testing.T, cost, price string) *domain.Product {
	t.Helper()
	p := validProductParams(t)
	p.Variant.Cost, p.Variant.Price = dec(cost), dec(price)
	product, err := domain.NewProduct(p, testNow)
	require.NoError(t, err)
	return product
}

func TestNewProduct(t *testing.T) {
	t.Run("crea producto activo con id v7", func(t *testing.T) {
		p := validProductParams(t)
		categoryID := uuid.Must(uuid.NewV7())
		p.CategoryID = &categoryID

		product, err := domain.NewProduct(p, testNow)
		require.NoError(t, err)
		require.Equal(t, 7, int(product.ID().Version()))
		require.Equal(t, p.TenantID, product.TenantID())
		require.Equal(t, "Cemento gris 50kg", product.Name())
		require.Equal(t, "Cemento de uso general", product.Description())
		require.Equal(t, &categoryID, product.CategoryID())
		require.Equal(t, domain.UnitBulto, product.BaseUnit())
		require.True(t, dec("0.19").Equal(product.TaxRate().Rate()))
		require.True(t, product.IsActive())
		require.Equal(t, testNow.UTC(), product.CreatedAt())
	})

	t.Run("un producto simple crea su variante por defecto", func(t *testing.T) {
		tests := []struct {
			name   string
			params func(*testing.T) domain.NewProductParams
			sku    string
		}{
			{"ferretería: cemento por bulto", validProductParams, "CEM-GRIS-50"},
			{"almacén de ropa: camiseta básica", func(t *testing.T) domain.NewProductParams {
				p := validProductParams(t)
				p.Name, p.BaseUnit = "Camiseta básica", domain.UnitPiece
				p.Variant.SKU, p.Variant.Barcode = "cam-basica", ""
				return p
			}, "CAM-BASICA"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := tt.params(t)
				product, err := domain.NewProduct(p, testNow)
				require.NoError(t, err)

				require.Len(t, product.Variants(), 1)
				v := product.DefaultVariant()
				require.Same(t, v, product.Variants()[0])
				require.True(t, v.IsDefault())
				require.True(t, v.IsActive())
				require.Equal(t, product.ID(), v.ProductID())
				require.Equal(t, tt.sku, v.SKU())
				require.Equal(t, strings.TrimSpace(p.Variant.Barcode), v.Barcode())
				require.True(t, p.Variant.Cost.Equal(v.Cost()))
				require.True(t, p.Variant.Price.Equal(v.Price()))
				require.Empty(t, v.AxisValues())
				require.Equal(t, testNow.UTC(), v.CreatedAt())
			})
		}
	})

	t.Run("un producto nuevo tiene una variante con un ID distinto al del producto", func(t *testing.T) {
		product, err := domain.NewProduct(validProductParams(t), testNow)
		require.NoError(t, err)

		v := product.DefaultVariant()
		require.Equal(t, 7, int(v.ID().Version()))
		require.NotEqual(t, product.ID(), v.ID())
	})

	t.Run("acepta producto sin código de barras ni categoría", func(t *testing.T) {
		p := validProductParams(t)
		p.Variant.Barcode = "   "
		p.CategoryID = nil

		product, err := domain.NewProduct(p, testNow)
		require.NoError(t, err)
		require.Empty(t, product.DefaultVariant().Barcode())
		require.Nil(t, product.CategoryID())
	})

	t.Run("acepta costo cero y precio con 4 decimales", func(t *testing.T) {
		p := validProductParams(t)
		p.Variant.Cost, p.Variant.Price = dec("0"), dec("1250.1234")

		_, err := domain.NewProduct(p, testNow)
		require.NoError(t, err)
	})

	t.Run("rechaza producto con datos inválidos", func(t *testing.T) {
		nilCategory := uuid.Nil
		tests := []struct {
			name   string
			mutate func(*domain.NewProductParams)
			want   error
		}{
			{"tenant vacío", func(p *domain.NewProductParams) { p.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"nombre vacío", func(p *domain.NewProductParams) { p.Name = "  " }, domain.ErrInvalidProductName},
			{"nombre demasiado largo", func(p *domain.NewProductParams) { p.Name = strings.Repeat("a", 201) }, domain.ErrInvalidProductName},
			{"descripción demasiado larga", func(p *domain.NewProductParams) { p.Description = strings.Repeat("a", 2001) }, domain.ErrInvalidDescription},
			{"categoría vacía", func(p *domain.NewProductParams) { p.CategoryID = &nilCategory }, domain.ErrInvalidCategoryID},
			{"unidad desconocida", func(p *domain.NewProductParams) { p.BaseUnit = "docena" }, domain.ErrInvalidUnit},
			{"SKU de la variante vacío", func(p *domain.NewProductParams) { p.Variant.SKU = "   " }, domain.ErrInvalidSKU},
			{"precio de la variante cero", func(p *domain.NewProductParams) { p.Variant.Price = dec("0") }, domain.ErrInvalidPrice},
			{"ejes de la variante con un valor vacío", func(p *domain.NewProductParams) {
				p.Variant.AxisValues = map[string]string{"color": " "}
			}, domain.ErrInvalidAxisValues},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := validProductParams(t)
				tt.mutate(&p)
				_, err := domain.NewProduct(p, testNow)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}

func TestProductActivation(t *testing.T) {
	t.Run("desactiva y reactiva un producto", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")

		product.Deactivate()
		product.Deactivate()
		require.False(t, product.IsActive())

		product.Activate()
		require.True(t, product.IsActive())
		require.NoError(t, product.ChangeVariantPrice(product.DefaultVariant().ID(), dec("12000")))
	})
}

func validSnapshot(t *testing.T) domain.ProductSnapshot {
	t.Helper()
	categoryID := uuid.Must(uuid.NewV7())
	id := uuid.Must(uuid.NewV7())
	return domain.ProductSnapshot{
		ID:         id,
		TenantID:   uuid.Must(uuid.NewV7()),
		Name:       "Tubo PVC presión 1/2 pulgada",
		CategoryID: &categoryID,
		BaseUnit:   domain.UnitMeter,
		TaxRate:    mustTaxRate(t, "0.19"),
		Active:     false,
		CreatedAt:  testNow,
		Variants: []domain.VariantSnapshot{{
			// A migrated product: its default variant reuses the product's id.
			ID:        id,
			SKU:       "TUB-PVC-12",
			Cost:      dec("3200.0000"),
			Price:     dec("4500.0000"),
			IsDefault: true,
			Active:    true,
			CreatedAt: testNow,
		}},
	}
}

func TestRehydrateProduct(t *testing.T) {
	t.Run("reconstruye un producto guardado con su variante por defecto", func(t *testing.T) {
		s := validSnapshot(t)
		product, err := domain.RehydrateProduct(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, product.ID())
		require.Equal(t, domain.UnitMeter, product.BaseUnit())
		require.False(t, product.IsActive())
		require.Equal(t, testNow.UTC(), product.CreatedAt())
		require.Equal(t, s.ID, product.DefaultVariant().ID())
		require.Equal(t, "TUB-PVC-12", product.DefaultVariant().SKU())
	})

	t.Run("reconstruye un producto con varias variantes", func(t *testing.T) {
		s := validSnapshot(t)
		s.Variants[0].AxisValues = map[string]string{"talla": "S", "color": "Rojo"}
		s.Variants = append(s.Variants, domain.VariantSnapshot{
			ID: uuid.Must(uuid.NewV7()), SKU: "CAM-M-ROJO", Cost: dec("15000"), Price: dec("30000"),
			AxisValues: map[string]string{"talla": "M", "color": "Rojo"}, Active: true, CreatedAt: testNow,
		})

		product, err := domain.RehydrateProduct(s)
		require.NoError(t, err)
		require.Len(t, product.Variants(), 2)
		require.Equal(t, s.Variants[0].ID, product.DefaultVariant().ID())
	})

	t.Run("rechaza datos inválidos al reconstruir", func(t *testing.T) {
		otherDefault := domain.VariantSnapshot{
			ID: uuid.Must(uuid.NewV7()), SKU: "OTRO", Cost: dec("1"), Price: dec("2"),
			AxisValues: map[string]string{"color": "Azul"}, IsDefault: true, Active: true, CreatedAt: testNow,
		}
		tests := []struct {
			name   string
			mutate func(*domain.ProductSnapshot)
			want   error
		}{
			{"id vacío", func(s *domain.ProductSnapshot) { s.ID = uuid.Nil }, domain.ErrInvalidProductID},
			{"tenant vacío", func(s *domain.ProductSnapshot) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"unidad desconocida", func(s *domain.ProductSnapshot) { s.BaseUnit = "" }, domain.ErrInvalidUnit},
			{"id de variante vacío", func(s *domain.ProductSnapshot) { s.Variants[0].ID = uuid.Nil }, domain.ErrInvalidVariantID},
			{"SKU vacío", func(s *domain.ProductSnapshot) { s.Variants[0].SKU = "" }, domain.ErrInvalidSKU},
			{"precio cero", func(s *domain.ProductSnapshot) { s.Variants[0].Price = dec("0") }, domain.ErrInvalidPrice},
			{"costo negativo", func(s *domain.ProductSnapshot) { s.Variants[0].Cost = dec("-1") }, domain.ErrInvalidCost},
			{"sin variantes", func(s *domain.ProductSnapshot) { s.Variants = nil }, domain.ErrInvalidVariants},
			{"sin variante por defecto", func(s *domain.ProductSnapshot) { s.Variants[0].IsDefault = false }, domain.ErrInvalidVariants},
			{"dos variantes por defecto", func(s *domain.ProductSnapshot) {
				s.Variants[0].AxisValues = map[string]string{"color": "Rojo"}
				s.Variants = append(s.Variants, otherDefault)
			}, domain.ErrInvalidVariants},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := validSnapshot(t)
				tt.mutate(&s)
				_, err := domain.RehydrateProduct(s)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}
