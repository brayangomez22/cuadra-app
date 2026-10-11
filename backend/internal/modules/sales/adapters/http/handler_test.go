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

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor   app.Actor
	confirm app.ConfirmSaleInput
	id      uuid.UUID
	reason  string
	list    app.ListSalesInput

	sale  *domain.Sale
	page  domain.SalePage
	limit domain.DiscountLimit
	err   error
}

func (s *stubUseCases) GetDiscountLimit(_ context.Context, a app.Actor) (domain.DiscountLimit, error) {
	s.actor = a
	return s.limit, s.err
}

func (s *stubUseCases) ConfirmSale(_ context.Context, a app.Actor, in app.ConfirmSaleInput) (*domain.Sale, error) {
	s.actor, s.confirm = a, in
	return s.sale, s.err
}

func (s *stubUseCases) VoidSale(_ context.Context, a app.Actor, id uuid.UUID, reason string) (*domain.Sale, error) {
	s.actor, s.id, s.reason = a, id, reason
	return s.sale, s.err
}

func (s *stubUseCases) GetSale(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Sale, error) {
	s.actor, s.id = a, id
	return s.sale, s.err
}

func (s *stubUseCases) ListSales(_ context.Context, a app.Actor, in app.ListSalesInput) (domain.SalePage, error) {
	s.actor, s.list = a, in
	return s.page, s.err
}

// discountOnlyOwner grants sales:discount only to owners.
func discountOnlyOwner(role, permission string) bool {
	return role == "owner" || permission != "sales:discount"
}

var owner = auth.Principal{TenantID: newID(), UserID: newID(), Role: "owner"}

func cashier() auth.Principal {
	p := owner
	p.Role = "cashier"
	return p
}

func serve(t *testing.T, uc *stubUseCases, p *auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, discountOnlyOwner, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if p != nil {
		req = req.WithContext(auth.WithPrincipal(req.Context(), *p))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// aSale is a stored sale: 2.5 × 8403.3613 at 19% with a discount of 100,
// plus 1 × 4500 at 0%, paid in cash with 30000.
func aSale(t *testing.T) *domain.Sale {
	t.Helper()
	s, err := domain.NewSale(domain.NewSaleParams{ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, CreatedAt: now})
	require.NoError(t, err)
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: newID(), ProductName: "Tubo PVC 1/2", Quantity: dec("2.5"), UnitPrice: dec("8403.3613"), TaxRate: dec("0.19"), Discount: dec("100"),
	}, domain.NoDiscountLimit))
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: newID(), ProductName: "Puntilla 2\"", Quantity: dec("1"), UnitPrice: dec("4500"), TaxRate: dec("0"), Discount: dec("0"),
	}, domain.NoDiscountLimit))
	cash, err := domain.NewPayment(domain.PaymentCash, dec("30000"))
	require.NoError(t, err)
	require.NoError(t, s.Confirm([]domain.Payment{cash}, now))
	require.NoError(t, s.AssignNumber(15))
	return s
}

type errorBody struct {
	Error struct {
		Code, Message string
		Details       map[string]string
	}
}

func TestConfirmSale(t *testing.T) {
	t.Run("traduce el carrito y responde la venta con montos en string", func(t *testing.T) {
		uc := &stubUseCases{sale: aSale(t)}
		product, location := newID(), newID()
		body := `{"location_id":"` + location.String() + `","lines":[` +
			`{"product_id":"` + product.String() + `","quantity":"2.5","discount":"100"},` +
			`{"product_id":"` + product.String() + `","quantity":"1"}],` +
			`"payments":[{"method":"cash","amount":"30000"},{"method":"nequi","amount":"1000.5"}]}`

		p := cashier()
		rec := serve(t, uc, &p, http.MethodPost, "/api/v1/sales", body)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID, UnlimitedDiscount: false}, uc.actor)
		require.Equal(t, location, uc.confirm.LocationID)
		require.Len(t, uc.confirm.Lines, 2)
		require.True(t, dec("100").Equal(uc.confirm.Lines[0].Discount))
		require.True(t, uc.confirm.Lines[1].Discount.IsZero(), "sin descuento es 0")
		require.Equal(t, "nequi", uc.confirm.Payments[1].Method)
		require.True(t, dec("1000.5").Equal(uc.confirm.Payments[1].Amount))

		var sale struct {
			Number int64
			Status string
			Paid   string
			Change string
			Lines  []map[string]any
			Totals struct {
				Subtotal, Discounts, Tax, Total string
				Taxes                           []map[string]string
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sale))
		require.Equal(t, int64(15), sale.Number)
		require.Equal(t, "confirmed", sale.Status)
		// 2.5 × 8403.3613 = 21008.40 − 100 = 20908.40; IVA 3972.60. + 4500.
		require.Equal(t, "25508.40", sale.Totals.Subtotal)
		require.Equal(t, "100.00", sale.Totals.Discounts)
		require.Equal(t, "3972.60", sale.Totals.Tax)
		require.Equal(t, "29381.00", sale.Totals.Total)
		require.Equal(t, "619.00", sale.Change)
		require.Equal(t, "8403.3613", sale.Lines[0]["unit_price"])
		require.Equal(t, "2.5", sale.Lines[0]["quantity"])
		require.Equal(t, "0.19", sale.Lines[0]["tax_rate"])
		require.Equal(t, "20908.40", sale.Lines[0]["base"])
		require.Equal(t, []map[string]string{
			{"rate": "0", "base": "4500.00", "tax": "0.00"},
			{"rate": "0.19", "base": "20908.40", "tax": "3972.60"},
		}, sale.Totals.Taxes)
		require.NotContains(t, rec.Body.String(), `"void"`)
	})

	t.Run("quien tiene sales:discount descuenta sin tope", func(t *testing.T) {
		uc := &stubUseCases{sale: aSale(t)}
		body := `{"location_id":"` + newID().String() + `","lines":[{"product_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/sales", body)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.True(t, uc.actor.UnlimitedDiscount)
	})

	t.Run("sin stock responde 409 con el nombre y el id del producto", func(t *testing.T) {
		product := newID()
		uc := &stubUseCases{err: &domain.InsufficientStockError{ProductID: product, ProductName: "Cemento gris 50kg"}}
		body := `{"location_id":"` + newID().String() + `","lines":[{"product_id":"` + product.String() + `","quantity":"1"}],"payments":[]}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/sales", body)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		var got errorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "insufficient_stock", got.Error.Code)
		require.Equal(t, "No hay stock suficiente de Cemento gris 50kg.", got.Error.Message)
		require.Equal(t, map[string]string{"product_id": product.String()}, got.Error.Details)
	})

	t.Run("mapea las reglas del dominio a su status y código", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrDiscountExceedsLimit, http.StatusForbidden, "discount_exceeds_limit"},
			{domain.ErrInsufficientPayment, http.StatusUnprocessableEntity, "insufficient_payment"},
			{domain.ErrNonCashOverpayment, http.StatusUnprocessableEntity, "non_cash_overpayment"},
			{domain.ErrProductInactive, http.StatusUnprocessableEntity, "product_inactive"},
			{domain.ErrLocationInactive, http.StatusUnprocessableEntity, "location_inactive"},
			{domain.ErrLocationNotFound, http.StatusUnprocessableEntity, "location_not_found"},
			{domain.ErrInsufficientStock, http.StatusConflict, "insufficient_stock"},
		}
		body := `{"location_id":"` + newID().String() + `","lines":[{"product_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`
		for _, tt := range tests {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPost, "/api/v1/sales", body)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"`+tt.code+`"`)
		}
	})

	t.Run("sin principal responde 401", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, nil, http.MethodPost, "/api/v1/sales", `{"location_id":"`+newID().String()+`","lines":[],"payments":[]}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestVoidSale(t *testing.T) {
	t.Run("anula y responde la venta con su anulación", func(t *testing.T) {
		sale := aSale(t)
		require.NoError(t, sale.Void(owner.UserID, "El cliente desistió", now))
		uc := &stubUseCases{sale: sale}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/sales/"+sale.ID().String()+"/void", `{"reason":"El cliente desistió"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, sale.ID(), uc.id)
		require.Equal(t, "El cliente desistió", uc.reason)
		require.Contains(t, rec.Body.String(), `"status":"voided"`)
		require.Contains(t, rec.Body.String(), `"void":{"reason":"El cliente desistió"`)
	})

	t.Run("una venta ya anulada responde 409 y una inexistente 404", func(t *testing.T) {
		path := "/api/v1/sales/" + newID().String() + "/void"
		rec := serve(t, &stubUseCases{err: domain.ErrSaleNotConfirmed}, &owner, http.MethodPost, path, `{"reason":"x"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Contains(t, rec.Body.String(), "sale_not_confirmed")

		rec = serve(t, &stubUseCases{err: domain.ErrSaleNotFound}, &owner, http.MethodPost, path, `{"reason":"x"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "sale_not_found")
	})
}

func TestGetSale(t *testing.T) {
	t.Run("responde el detalle de la venta", func(t *testing.T) {
		sale := aSale(t)
		uc := &stubUseCases{sale: sale}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/sales/"+sale.ID().String(), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, sale.ID(), uc.id)
		require.Contains(t, rec.Body.String(), `"number":15`)
	})
}

func TestListSales(t *testing.T) {
	t.Run("pasa el día, la sede y la página y responde los resúmenes", func(t *testing.T) {
		location := newID()
		voidedAt := now.Add(time.Hour)
		uc := &stubUseCases{page: domain.SalePage{Total: 1, Items: []domain.SaleSummary{{
			ID: newID(), Number: 3, LocationID: location, UserID: owner.UserID, Status: domain.SaleVoided,
			Total: dec("35700.0000"), ConfirmedAt: now, VoidedAt: &voidedAt,
		}}}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/sales?date=2026-10-09&location_id="+location.String()+"&limit=5&offset=10", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "2026-10-09", uc.list.Date.Format(time.DateOnly))
		require.Equal(t, &location, uc.list.LocationID)
		require.Equal(t, 5, uc.list.Limit)
		require.Equal(t, 10, uc.list.Offset)
		require.Contains(t, rec.Body.String(), `"total":"35700.00"`)
		require.Contains(t, rec.Body.String(), `"status":"voided"`)
	})

	t.Run("sin parámetros pide hoy con la página por defecto", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/sales", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.True(t, uc.list.Date.IsZero())
		require.Nil(t, uc.list.LocationID)
		require.Equal(t, 20, uc.list.Limit)
		require.Contains(t, rec.Body.String(), `"items":[]`)
	})
}

func TestGetDiscountLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit domain.DiscountLimit
		want  string
	}{
		{"responde el tope del usuario autenticado", mustLimit(t, "0.1"), `{"unlimited":false,"max":"0.1"}`},
		{"sin tope responde unlimited y max 1", domain.NoDiscountLimit, `{"unlimited":true,"max":"1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCases{limit: tt.limit}
			p := cashier()
			rec := serve(t, uc, &p, http.MethodGet, "/api/v1/sales/discount-limit", "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, tt.want, rec.Body.String())
			require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID}, uc.actor)
		})
	}
}

func mustLimit(t *testing.T, fraction string) domain.DiscountLimit {
	t.Helper()
	l, err := domain.NewDiscountLimit(dec(fraction))
	require.NoError(t, err)
	return l
}
