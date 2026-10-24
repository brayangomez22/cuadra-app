package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestCreateProduct(t *testing.T) {
	t.Run("crear producto devuelve 201 con el dinero como string", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		body := decodeMap(t, rec)
		variant := body["variants"].([]any)[0].(map[string]any)
		require.Equal(t, "TUB-PVC-12", variant["sku"])
		require.Equal(t, "10000.00", variant["price"])
		require.Equal(t, "8333.33", variant["cost"])
		require.Equal(t, "11900.00", variant["price_with_tax"])
		require.Equal(t, "16.6667", variant["margin_percent"])
		require.Equal(t, "0.19", body["tax_rate"])
		require.Equal(t, meters.ID().String(), body["base_unit_id"])
		for _, removed := range []string{"sku", "barcode", "base_unit", "cost", "price", "price_with_tax", "margin_percent"} {
			require.NotContains(t, body, removed)
		}
		require.Equal(t, map[string]any{"id": meters.ID().String(), "code": "m", "name": "Metro"}, body["unit"])
		require.Equal(t, true, body["active"])
		require.NotContains(t, body, "category_id")
		require.NotContains(t, body, "description")
	})

	t.Run("pasa los datos y el actor del token al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}

		serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody)

		require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID}, uc.actor)
		require.Equal(t, "tub-pvc-12", uc.input.SKU)
		require.Equal(t, "7701234567890", uc.input.Barcode)
		require.Equal(t, meters.ID(), uc.input.BaseUnitID)
		require.Equal(t, map[string]domain.AttributeValue{}, uc.input.Attributes)
		require.True(t, decimal.RequireFromString("8333.33").Equal(uc.input.Cost))
		require.True(t, decimal.RequireFromString("10000").Equal(uc.input.Price))
		require.True(t, decimal.RequireFromString("0.19").Equal(uc.input.TaxRate.Rate()))
		require.Nil(t, uc.input.CategoryID)
	})

	t.Run("errores de la unidad del producto", func(t *testing.T) {
		tests := []struct {
			err  error
			code string
		}{
			{domain.ErrUnitNotFound, "unit_not_found"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				uc := &stubUseCases{err: tt.err}
				rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody)
				requireError(t, rec, http.StatusUnprocessableEntity, tt.code)
			})
		}
	})

	t.Run("muestra hasta 4 decimales cuando el monto los tiene", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.3333", "10000.5")}

		body := decodeMap(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody))
		variant := body["variants"].([]any)[0].(map[string]any)

		require.Equal(t, "8333.3333", variant["cost"])
		require.Equal(t, "10000.50", variant["price"])
	})

	t.Run("SKU duplicado responde 409 sku_taken", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrSKUTaken}
		requireError(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody), http.StatusConflict, "sku_taken")
	})

	t.Run("categoría inexistente responde 422 category_not_found", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCategoryNotFound}
		requireError(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody), http.StatusUnprocessableEntity, "category_not_found")
	})

	t.Run("precio inválido para el dominio responde 422 invalid_price", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrInvalidPrice}
		requireError(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody), http.StatusUnprocessableEntity, "invalid_price")
	})

	t.Run("tasa de IVA de 1 o más responde 422 invalid_tax_rate sin llamar al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		body := strings.Replace(productBody, `"tax_rate":"0.19"`, `"tax_rate":"1"`, 1)

		requireError(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", body), http.StatusUnprocessableEntity, "invalid_tax_rate")
		require.Empty(t, uc.input.SKU)
	})

	t.Run("JSON mal formado responde 400 invalid_request", func(t *testing.T) {
		requireError(t, serve(t, &stubUseCases{}, owner, http.MethodPost, "/api/v1/products", `{"sku":`), http.StatusBadRequest, "invalid_request")
	})
}

func TestGetProduct(t *testing.T) {
	t.Run("el cajero no ve el costo ni el margen", func(t *testing.T) {
		p := product(t, "8333.33", "10000")
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/products/"+p.ID().String(), "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		variant := decodeMap(t, rec)["variants"].([]any)[0].(map[string]any)
		require.Equal(t, "10000.00", variant["price"])
		require.NotContains(t, variant, "cost")
		require.NotContains(t, variant, "margin_percent")
		require.Equal(t, p.ID(), uc.id)
	})

	t.Run("producto inexistente responde 404 product_not_found", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrProductNotFound}
		requireError(t, serve(t, uc, owner, http.MethodGet, "/api/v1/products/"+uuid.NewString(), ""), http.StatusNotFound, "product_not_found")
	})
}

func TestUpdateProduct(t *testing.T) {
	t.Run("cambiar el precio de un producto inactivo responde 422 product_inactive", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrProductInactive}
		id := uuid.Must(uuid.NewV7())

		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/products/"+id.String(), productBody)

		requireError(t, rec, http.StatusUnprocessableEntity, "product_inactive")
		require.Equal(t, id, uc.id)
	})

	t.Run("desactiva y activa un producto", func(t *testing.T) {
		p := product(t, "1", "2")
		p.Deactivate()
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products/"+p.ID().String()+"/deactivate", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, false, decodeMap(t, rec)["active"])

		p.Activate()
		rec = serve(t, uc, owner, http.MethodPost, "/api/v1/products/"+p.ID().String()+"/activate", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, true, decodeMap(t, rec)["active"])
	})
}

func TestListProducts(t *testing.T) {
	t.Run("lista activos con la paginación por defecto", func(t *testing.T) {
		uc := &stubUseCases{page: domain.ProductPage{Items: []*domain.Product{product(t, "1", "2")}, Total: 7}}

		rec := serve(t, uc, owner, http.MethodGet, "/api/v1/products?q=tubo+pvc", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Items, 1)
		require.Equal(t, 7, body.Total)
		require.Equal(t, "tubo pvc", uc.filter.Query)
		require.NotNil(t, uc.filter.Active)
		require.True(t, *uc.filter.Active)
		require.Equal(t, 20, uc.filter.Limit)
		require.Zero(t, uc.filter.Offset)
	})

	t.Run("pasa estado, categoría y paginación", func(t *testing.T) {
		uc := &stubUseCases{}
		categoryID := uuid.Must(uuid.NewV7())

		rec := serve(t, uc, owner, http.MethodGet, "/api/v1/products?status=all&category_id="+categoryID.String()+"&limit=50&offset=100", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Nil(t, uc.filter.Active)
		require.Equal(t, &categoryID, uc.filter.CategoryID)
		require.Equal(t, 50, uc.filter.Limit)
		require.Equal(t, 100, uc.filter.Offset)
		require.Contains(t, rec.Body.String(), `"items":[]`)
	})

	t.Run("pasa los ids pedidos al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		first, second := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

		rec := serve(t, uc, owner, http.MethodGet, "/api/v1/products?status=all&ids="+first.String()+"&ids="+second.String(), "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, []uuid.UUID{first, second}, uc.filter.IDs)
	})

	t.Run("sin ids no filtra por id", func(t *testing.T) {
		uc := &stubUseCases{}
		serve(t, uc, owner, http.MethodGet, "/api/v1/products", "")
		require.Nil(t, uc.filter.IDs)
	})

	t.Run("status inactive filtra los inactivos", func(t *testing.T) {
		uc := &stubUseCases{}
		serve(t, uc, owner, http.MethodGet, "/api/v1/products?status=inactive", "")
		require.NotNil(t, uc.filter.Active)
		require.False(t, *uc.filter.Active)
	})
}
