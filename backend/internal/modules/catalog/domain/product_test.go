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
		SKU:         "  cem-gris-50  ",
		Barcode:     "7701234567890",
		Name:        "  Cemento gris 50kg  ",
		Description: "Cemento de uso general",
		BaseUnit:    domain.UnitBulto,
		Cost:        dec("28000"),
		Price:       dec("32500.50"),
		TaxRate:     mustTaxRate(t, "0.19"),
	}
}

func mustProduct(t *testing.T, cost, price string) *domain.Product {
	t.Helper()
	p := validProductParams(t)
	p.Cost, p.Price = dec(cost), dec(price)
	product, err := domain.NewProduct(p, testNow)
	require.NoError(t, err)
	return product
}

func TestNewProduct(t *testing.T) {
	t.Run("crea producto activo con id v7 y SKU normalizado", func(t *testing.T) {
		p := validProductParams(t)
		categoryID := uuid.Must(uuid.NewV7())
		p.CategoryID = &categoryID

		product, err := domain.NewProduct(p, testNow)
		require.NoError(t, err)
		require.Equal(t, 7, int(product.ID().Version()))
		require.Equal(t, p.TenantID, product.TenantID())
		require.Equal(t, "CEM-GRIS-50", product.SKU())
		require.Equal(t, "7701234567890", product.Barcode())
		require.Equal(t, "Cemento gris 50kg", product.Name())
		require.Equal(t, "Cemento de uso general", product.Description())
		require.Equal(t, &categoryID, product.CategoryID())
		require.Equal(t, domain.UnitBulto, product.BaseUnit())
		require.True(t, dec("28000").Equal(product.Cost()))
		require.True(t, dec("32500.50").Equal(product.Price()))
		require.True(t, dec("0.19").Equal(product.TaxRate().Rate()))
		require.True(t, product.IsActive())
		require.Equal(t, testNow.UTC(), product.CreatedAt())
	})

	t.Run("acepta producto sin código de barras ni categoría", func(t *testing.T) {
		p := validProductParams(t)
		p.Barcode = "   "
		p.CategoryID = nil

		product, err := domain.NewProduct(p, testNow)
		require.NoError(t, err)
		require.Empty(t, product.Barcode())
		require.Nil(t, product.CategoryID())
	})

	t.Run("acepta costo cero y precio con 4 decimales", func(t *testing.T) {
		p := validProductParams(t)
		p.Cost, p.Price = dec("0"), dec("1250.1234")

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
			{"SKU vacío", func(p *domain.NewProductParams) { p.SKU = "   " }, domain.ErrInvalidSKU},
			{"SKU con espacios internos", func(p *domain.NewProductParams) { p.SKU = "CEM GRIS" }, domain.ErrInvalidSKU},
			{"SKU demasiado largo", func(p *domain.NewProductParams) { p.SKU = strings.Repeat("A", 65) }, domain.ErrInvalidSKU},
			{"código de barras con espacios", func(p *domain.NewProductParams) { p.Barcode = "770 123" }, domain.ErrInvalidBarcode},
			{"código de barras no ASCII", func(p *domain.NewProductParams) { p.Barcode = "77012ñ" }, domain.ErrInvalidBarcode},
			{"nombre vacío", func(p *domain.NewProductParams) { p.Name = "  " }, domain.ErrInvalidProductName},
			{"nombre demasiado largo", func(p *domain.NewProductParams) { p.Name = strings.Repeat("a", 201) }, domain.ErrInvalidProductName},
			{"descripción demasiado larga", func(p *domain.NewProductParams) { p.Description = strings.Repeat("a", 2001) }, domain.ErrInvalidDescription},
			{"categoría vacía", func(p *domain.NewProductParams) { p.CategoryID = &nilCategory }, domain.ErrInvalidCategoryID},
			{"unidad desconocida", func(p *domain.NewProductParams) { p.BaseUnit = "docena" }, domain.ErrInvalidUnit},
			{"precio cero", func(p *domain.NewProductParams) { p.Price = dec("0") }, domain.ErrInvalidPrice},
			{"precio negativo", func(p *domain.NewProductParams) { p.Price = dec("-100") }, domain.ErrInvalidPrice},
			{"precio con más de 4 decimales", func(p *domain.NewProductParams) { p.Price = dec("100.12345") }, domain.ErrInvalidPrice},
			{"precio fuera de rango", func(p *domain.NewProductParams) { p.Price = dec("100000000000000") }, domain.ErrInvalidPrice},
			{"costo negativo", func(p *domain.NewProductParams) { p.Cost = dec("-0.01") }, domain.ErrInvalidCost},
			{"costo con más de 4 decimales", func(p *domain.NewProductParams) { p.Cost = dec("0.00001") }, domain.ErrInvalidCost},
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

func TestProductMargin(t *testing.T) {
	t.Run("calcula el margen sin pérdida de precisión", func(t *testing.T) {
		product := mustProduct(t, "8333.33", "10000")
		require.Equal(t, "16.6667", product.MarginPercent().String())
	})

	t.Run("redondea el margen half-up a 4 decimales", func(t *testing.T) {
		// (3 - 1) / 3 × 100 = 66.66666…
		product := mustProduct(t, "1", "3")
		require.Equal(t, "66.6667", product.MarginPercent().String())
	})

	t.Run("margen es 100 cuando el costo es cero", func(t *testing.T) {
		product := mustProduct(t, "0", "5000")
		require.True(t, dec("100").Equal(product.MarginPercent()))
	})

	t.Run("margen es negativo cuando se vende por debajo del costo", func(t *testing.T) {
		product := mustProduct(t, "12000", "10000")
		require.True(t, dec("-20").Equal(product.MarginPercent()))
	})
}

func TestProductPriceWithTax(t *testing.T) {
	tests := []struct {
		name, price, rate, want string
	}{
		{"suma el IVA del 19% y redondea a 2 decimales", "8403.36", "0.19", "10000.00"},
		{"redondea half-up", "10.0250", "0", "10.03"},
		{"suma el IVA del 5%", "1000", "0.05", "1050.00"},
		{"producto exento queda igual", "2500", "0", "2500.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProductParams(t)
			p.Price, p.TaxRate = dec(tt.price), mustTaxRate(t, tt.rate)
			product, err := domain.NewProduct(p, testNow)
			require.NoError(t, err)
			require.Equal(t, tt.want, product.PriceWithTax().StringFixed(2))
		})
	}
}

func TestProductChangePrice(t *testing.T) {
	t.Run("cambia el precio de un producto activo", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		require.NoError(t, product.ChangePrice(dec("11500.25")))
		require.True(t, dec("11500.25").Equal(product.Price()))
	})

	t.Run("rechaza cambio a precio negativo o cero", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		for _, price := range []string{"0", "-1", "10.00001"} {
			require.ErrorIs(t, product.ChangePrice(dec(price)), domain.ErrInvalidPrice)
		}
		require.True(t, dec("10000").Equal(product.Price()))
	})

	t.Run("producto inactivo no puede cambiar de precio", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		product.Deactivate()

		require.ErrorIs(t, product.ChangePrice(dec("12000")), domain.ErrProductInactive)
		require.True(t, dec("10000").Equal(product.Price()))
	})

	t.Run("desactiva y reactiva un producto", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")

		product.Deactivate()
		product.Deactivate()
		require.False(t, product.IsActive())

		product.Activate()
		require.True(t, product.IsActive())
		require.NoError(t, product.ChangePrice(dec("12000")))
	})
}

func TestRehydrateProduct(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV7())
	valid := func() domain.ProductSnapshot {
		return domain.ProductSnapshot{
			ID:         uuid.Must(uuid.NewV7()),
			TenantID:   uuid.Must(uuid.NewV7()),
			SKU:        "TUB-PVC-12",
			Barcode:    "",
			Name:       "Tubo PVC presión 1/2 pulgada",
			CategoryID: &categoryID,
			BaseUnit:   domain.UnitMeter,
			Cost:       dec("3200.0000"),
			Price:      dec("4500.0000"),
			TaxRate:    mustTaxRate(t, "0.19"),
			Active:     false,
			CreatedAt:  testNow,
		}
	}

	t.Run("reconstruye un producto guardado", func(t *testing.T) {
		s := valid()
		product, err := domain.RehydrateProduct(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, product.ID())
		require.Equal(t, "TUB-PVC-12", product.SKU())
		require.Equal(t, domain.UnitMeter, product.BaseUnit())
		require.False(t, product.IsActive())
		require.Equal(t, testNow.UTC(), product.CreatedAt())
	})

	t.Run("rechaza datos inválidos al reconstruir", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*domain.ProductSnapshot)
			want   error
		}{
			{"id vacío", func(s *domain.ProductSnapshot) { s.ID = uuid.Nil }, domain.ErrInvalidProductID},
			{"tenant vacío", func(s *domain.ProductSnapshot) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"SKU vacío", func(s *domain.ProductSnapshot) { s.SKU = "" }, domain.ErrInvalidSKU},
			{"precio cero", func(s *domain.ProductSnapshot) { s.Price = dec("0") }, domain.ErrInvalidPrice},
			{"costo negativo", func(s *domain.ProductSnapshot) { s.Cost = dec("-1") }, domain.ErrInvalidCost},
			{"unidad desconocida", func(s *domain.ProductSnapshot) { s.BaseUnit = "" }, domain.ErrInvalidUnit},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := valid()
				tt.mutate(&s)
				_, err := domain.RehydrateProduct(s)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}
