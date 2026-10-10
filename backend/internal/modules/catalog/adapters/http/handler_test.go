package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor    app.Actor
	input    app.ProductInput
	filter   domain.ProductFilter
	id       uuid.UUID
	name     string
	parentID *uuid.UUID

	product  *domain.Product
	page     domain.ProductPage
	category *domain.Category
	list     []*domain.Category
	err      error
}

func (s *stubUseCases) CreateProduct(_ context.Context, a app.Actor, in app.ProductInput) (*domain.Product, error) {
	s.actor, s.input = a, in
	return s.product, s.err
}

func (s *stubUseCases) UpdateProduct(_ context.Context, a app.Actor, id uuid.UUID, in app.ProductInput) (*domain.Product, error) {
	s.actor, s.id, s.input = a, id, in
	return s.product, s.err
}

func (s *stubUseCases) GetProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) SearchProducts(_ context.Context, a app.Actor, f domain.ProductFilter) (domain.ProductPage, error) {
	s.actor, s.filter = a, f
	return s.page, s.err
}

func (s *stubUseCases) DeactivateProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) ActivateProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) CreateCategory(_ context.Context, a app.Actor, name string, parentID *uuid.UUID) (*domain.Category, error) {
	s.actor, s.name, s.parentID = a, name, parentID
	return s.category, s.err
}

func (s *stubUseCases) UpdateCategory(_ context.Context, a app.Actor, id uuid.UUID, name string, parentID *uuid.UUID) (*domain.Category, error) {
	s.actor, s.id, s.name, s.parentID = a, id, name, parentID
	return s.category, s.err
}

func (s *stubUseCases) ListCategories(_ context.Context, a app.Actor) ([]*domain.Category, error) {
	s.actor = a
	return s.list, s.err
}

func (s *stubUseCases) DeleteCategory(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

// canWrite grants catalog:write only to owners, like a reduced permission matrix.
func canWrite(role, permission string) bool { return role == "owner" || permission == "catalog:read" }

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func cashier() auth.Principal {
	p := owner
	p.Role = "cashier"
	return p
}

func serve(t *testing.T, uc *stubUseCases, p auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, canWrite, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(auth.WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m), rec.Body.String())
	return m
}

func product(t *testing.T, cost, price string) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(decimal.RequireFromString("0.19"))
	require.NoError(t, err)
	p, err := domain.NewProduct(domain.NewProductParams{
		TenantID: owner.TenantID, SKU: "TUB-PVC-12", Barcode: "7701234567890", Name: "Tubo PVC presión 1/2 pulgada",
		BaseUnit: domain.UnitMeter, Cost: decimal.RequireFromString(cost), Price: decimal.RequireFromString(price), TaxRate: rate,
	}, now)
	require.NoError(t, err)
	return p
}

const productBody = `{"sku":"tub-pvc-12","barcode":"7701234567890","name":"Tubo PVC presión 1/2 pulgada","base_unit":"m","cost":"8333.33","price":"10000","tax_rate":"0.19"}`

func TestCreateProduct(t *testing.T) {
	t.Run("crear producto devuelve 201 con el dinero como string", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		body := decodeMap(t, rec)
		require.Equal(t, "TUB-PVC-12", body["sku"])
		require.Equal(t, "10000.00", body["price"])
		require.Equal(t, "8333.33", body["cost"])
		require.Equal(t, "11900.00", body["price_with_tax"])
		require.Equal(t, "0.19", body["tax_rate"])
		require.Equal(t, "16.6667", body["margin_percent"])
		require.Equal(t, "m", body["base_unit"])
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
		require.Equal(t, domain.UnitMeter, uc.input.BaseUnit)
		require.True(t, decimal.RequireFromString("8333.33").Equal(uc.input.Cost))
		require.True(t, decimal.RequireFromString("10000").Equal(uc.input.Price))
		require.True(t, decimal.RequireFromString("0.19").Equal(uc.input.TaxRate.Rate()))
		require.Nil(t, uc.input.CategoryID)
	})

	t.Run("muestra hasta 4 decimales cuando el monto los tiene", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.3333", "10000.5")}

		body := decodeMap(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody))

		require.Equal(t, "8333.3333", body["cost"])
		require.Equal(t, "10000.50", body["price"])
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
		body := decodeMap(t, rec)
		require.Equal(t, "10000.00", body["price"])
		require.NotContains(t, body, "cost")
		require.NotContains(t, body, "margin_percent")
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

	t.Run("status inactive filtra los inactivos", func(t *testing.T) {
		uc := &stubUseCases{}
		serve(t, uc, owner, http.MethodGet, "/api/v1/products?status=inactive", "")
		require.NotNil(t, uc.filter.Active)
		require.False(t, *uc.filter.Active)
	})
}

func TestCategories(t *testing.T) {
	t.Run("crea una categoría con padre", func(t *testing.T) {
		parentID := uuid.Must(uuid.NewV7())
		c, err := domain.NewCategory(owner.TenantID, "Tubería", &parentID, now)
		require.NoError(t, err)
		uc := &stubUseCases{category: c}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/categories", `{"name":"Tubería","parent_id":"`+parentID.String()+`"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		body := decodeMap(t, rec)
		require.Equal(t, "Tubería", body["name"])
		require.Equal(t, parentID.String(), body["parent_id"])
		require.Equal(t, "Tubería", uc.name)
		require.Equal(t, &parentID, uc.parentID)
	})

	t.Run("lista las categorías", func(t *testing.T) {
		c, err := domain.NewCategory(owner.TenantID, "Plomería", nil, now)
		require.NoError(t, err)
		uc := &stubUseCases{list: []*domain.Category{c}}

		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/categories", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"name":"Plomería"`)
		require.NotContains(t, rec.Body.String(), "parent_id")
	})

	t.Run("categoría con ciclo responde 422 category_cycle", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCategoryCycle}
		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/categories/"+uuid.NewString(), `{"name":"A","parent_id":"`+uuid.NewString()+`"}`)
		requireError(t, rec, http.StatusUnprocessableEntity, "category_cycle")
	})

	t.Run("borrar categoría en uso responde 409 category_in_use", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCategoryInUse}
		requireError(t, serve(t, uc, owner, http.MethodDelete, "/api/v1/categories/"+uuid.NewString(), ""), http.StatusConflict, "category_in_use")
	})

	t.Run("borrar categoría vacía responde 204", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		uc := &stubUseCases{}

		rec := serve(t, uc, owner, http.MethodDelete, "/api/v1/categories/"+id.String(), "")

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
	})
}

func TestMissingPrincipal(t *testing.T) {
	t.Run("sin principal responde 401 y no llama al caso de uso", func(t *testing.T) {
		mux := http.NewServeMux()
		uc := &stubUseCases{}
		httpadapter.Register(mux, uc, canWrite, slog.New(slog.DiscardHandler))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.Equal(t, app.Actor{}, uc.actor)
	})
}
