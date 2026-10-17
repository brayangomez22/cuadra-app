package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// productWithVariants returns a garment with two variants: S (the default)
// and M.
func productWithVariants(t *testing.T) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(decimal.RequireFromString("0.19"))
	require.NoError(t, err)
	p, err := domain.NewProduct(domain.NewProductParams{
		TenantID: owner.TenantID, Name: "Camiseta básica", BaseUnit: mustUnit("und", "Unidad").Ref(), TaxRate: rate,
		Variant: domain.VariantParams{
			SKU: "CAM-S", Barcode: "7701111111111", Cost: decimal.RequireFromString("15000"),
			Price: decimal.RequireFromString("25000"), AxisValues: map[string]string{"talla": "S"},
		},
	}, now)
	require.NoError(t, err)
	_, err = p.AddVariant(domain.VariantParams{
		SKU: "CAM-M", Cost: decimal.RequireFromString("16000"), Price: decimal.RequireFromString("26000.5"),
		AxisValues: map[string]string{"talla": "M"},
	}, now)
	require.NoError(t, err)
	return p
}

type variantJSON struct {
	ID            string            `json:"id"`
	SKU           string            `json:"sku"`
	Barcode       *string           `json:"barcode"`
	Price         string            `json:"price"`
	PriceWithTax  string            `json:"price_with_tax"`
	Cost          *string           `json:"cost"`
	MarginPercent *string           `json:"margin_percent"`
	AxisValues    map[string]string `json:"axis_values"`
	Label         string            `json:"label"`
	IsDefault     bool              `json:"is_default"`
	Active        bool              `json:"active"`
	CreatedAt     string            `json:"created_at"`
}

type productJSON struct {
	ID           string        `json:"id"`
	SKU          string        `json:"sku"`
	Barcode      *string       `json:"barcode"`
	Price        string        `json:"price"`
	PriceWithTax string        `json:"price_with_tax"`
	Cost         *string       `json:"cost"`
	Variants     []variantJSON `json:"variants"`
}

func decodeProduct(t *testing.T, raw []byte) productJSON {
	t.Helper()
	var p productJSON
	require.NoError(t, json.Unmarshal(raw, &p), string(raw))
	return p
}

func TestProductVariantsBody(t *testing.T) {
	t.Run("el producto responde variants y los campos deprecated iguales a su variante por defecto", func(t *testing.T) {
		p := productWithVariants(t)
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, owner, http.MethodGet, "/api/v1/products/"+p.ID().String(), "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decodeProduct(t, rec.Body.Bytes())
		require.Len(t, body.Variants, 2)
		def, m := body.Variants[0], body.Variants[1]
		require.Equal(t, p.DefaultVariant().ID().String(), def.ID)
		require.True(t, def.IsDefault)
		require.Equal(t, map[string]string{"talla": "S"}, def.AxisValues)
		require.Equal(t, "CAM-S", body.SKU)
		require.Equal(t, def.SKU, body.SKU)
		require.Equal(t, def.Barcode, body.Barcode)
		require.Equal(t, def.Price, body.Price)
		require.Equal(t, def.PriceWithTax, body.PriceWithTax)
		require.Equal(t, def.Cost, body.Cost)

		require.Equal(t, "CAM-M", m.SKU)
		require.False(t, m.IsDefault)
		require.True(t, m.Active)
		require.Nil(t, m.Barcode)
		require.Equal(t, "26000.50", m.Price)
		require.Equal(t, "30940.60", m.PriceWithTax)
		require.Equal(t, "16000.00", *m.Cost)
		require.Equal(t, "38.4627", *m.MarginPercent)
		require.Equal(t, map[string]string{"talla": "M"}, m.AxisValues)
		require.NotEmpty(t, m.CreatedAt)
	})

	t.Run("cada variante responde su label: los valores ordenados por nombre de eje, vacío sin ejes", func(t *testing.T) {
		tests := []struct {
			name    string
			product *domain.Product
			want    []string
		}{
			{name: "ropa: un eje", product: productWithVariants(t), want: []string{"S", "M"}},
			{name: "ropa: dos ejes, ordenados por nombre", product: twoAxesProduct(t), want: []string{"Rojo · M", "Azul · L"}},
			{name: "ferretería: producto simple", product: product(t, "8000", "10000"), want: []string{""}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rec := serve(t, &stubUseCases{product: tt.product}, owner, http.MethodGet, "/api/v1/products/"+tt.product.ID().String(), "")

				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				var labels []string
				for _, v := range decodeProduct(t, rec.Body.Bytes()).Variants {
					labels = append(labels, v.Label)
				}
				require.Equal(t, tt.want, labels)
			})
		}
	})

	t.Run("el costo de las variantes solo se envía con catalog:write", func(t *testing.T) {
		p := productWithVariants(t)

		rec := serve(t, &stubUseCases{product: p}, cashier(), http.MethodGet, "/api/v1/products/"+p.ID().String(), "")

		body := decodeProduct(t, rec.Body.Bytes())
		for _, v := range body.Variants {
			require.Nil(t, v.Cost)
			require.Nil(t, v.MarginPercent)
		}
		require.Nil(t, body.Cost)
	})
}

const variantBody = `{"sku":"cam-l","barcode":"7702222222222","cost":"17000","price":"27000","axis_values":{"talla":"L"}}`

func TestAddVariant(t *testing.T) {
	t.Run("agrega una variante: 201 con el producto completo", func(t *testing.T) {
		p := productWithVariants(t)
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products/"+p.ID().String()+"/variants", variantBody)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Len(t, decodeProduct(t, rec.Body.Bytes()).Variants, 2)
		require.Equal(t, p.ID(), uc.id)
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, "cam-l", uc.variant.SKU)
		require.Equal(t, "7702222222222", uc.variant.Barcode)
		require.True(t, decimal.RequireFromString("17000").Equal(uc.variant.Cost))
		require.True(t, decimal.RequireFromString("27000").Equal(uc.variant.Price))
		require.Equal(t, map[string]string{"talla": "L"}, uc.variant.AxisValues)
	})

	t.Run("mapea los errores de las variantes", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrDuplicateVariant, http.StatusUnprocessableEntity, "duplicate_variant"},
			{domain.ErrVariantAxesMismatch, http.StatusUnprocessableEntity, "variant_axes_mismatch"},
			{domain.ErrInvalidAxisValues, http.StatusUnprocessableEntity, "invalid_axis_values"},
			{domain.ErrSKUTaken, http.StatusConflict, "sku_taken"},
			{domain.ErrProductNotFound, http.StatusNotFound, "product_not_found"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				uc := &stubUseCases{err: tt.err}
				rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products/"+uuid.NewString()+"/variants", variantBody)
				requireError(t, rec, tt.status, tt.code)
			})
		}
	})
}

func TestUpdateVariant(t *testing.T) {
	t.Run("actualiza una variante y pasa su estado", func(t *testing.T) {
		p := productWithVariants(t)
		v := p.Variants()[1]
		uc := &stubUseCases{product: p}
		body := `{"sku":"cam-m","cost":"16000","price":"26500","axis_values":{"talla":"M"},"active":false}`

		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/products/"+p.ID().String()+"/variants/"+v.ID().String(), body)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.ID(), uc.id)
		require.Equal(t, v.ID(), uc.variantID)
		require.Equal(t, "cam-m", uc.variant.SKU)
		require.Empty(t, uc.variant.Barcode)
		require.True(t, decimal.RequireFromString("26500").Equal(uc.variant.Price))
		require.False(t, uc.variant.Active)
	})

	t.Run("una variante inexistente responde 404 variant_not_found", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrVariantNotFound}
		body := `{"sku":"cam-m","cost":"1","price":"2","axis_values":{},"active":true}`
		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/products/"+uuid.NewString()+"/variants/"+uuid.NewString(), body)
		requireError(t, rec, http.StatusNotFound, "variant_not_found")
	})
}

func TestPOSOperation(t *testing.T) {
	t.Run("la búsqueda del POS registra cuadra.operation=pos.search", func(t *testing.T) {
		_, labeler := serveLabeled(t, &stubUseCases{}, cashier(), http.MethodGet, "/api/v1/products?q=7701234567890", "")
		require.Contains(t, labeler.Get(), attribute.String("cuadra.operation", "pos.search"))
	})

	t.Run("un listado sin texto no cuenta como búsqueda del POS", func(t *testing.T) {
		_, labeler := serveLabeled(t, &stubUseCases{}, owner, http.MethodGet, "/api/v1/products?status=all", "")
		require.Empty(t, labeler.Get())
	})
}

// twoAxesProduct returns a garment that varies on talla and color, with its
// axes named out of alphabetical order.
func twoAxesProduct(t *testing.T) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(decimal.RequireFromString("0.19"))
	require.NoError(t, err)
	p, err := domain.NewProduct(domain.NewProductParams{
		TenantID: owner.TenantID, Name: "Camisa", BaseUnit: mustUnit("und", "Unidad").Ref(), TaxRate: rate,
		Variant: domain.VariantParams{
			SKU: "CAMISA-M-ROJO", Cost: decimal.RequireFromString("1"), Price: decimal.RequireFromString("2"),
			AxisValues: map[string]string{"talla": "M", "color": "Rojo"},
		},
	}, now)
	require.NoError(t, err)
	_, err = p.AddVariant(domain.VariantParams{
		SKU: "CAMISA-L-AZUL", Cost: decimal.RequireFromString("1"), Price: decimal.RequireFromString("2"),
		AxisValues: map[string]string{"talla": "L", "color": "Azul"},
	}, now)
	require.NoError(t, err)
	return p
}
