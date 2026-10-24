package httpadapter_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

func TestGetMe(t *testing.T) {
	t.Run("devuelve el usuario del token con su empresa, su NIT, sus permisos y los módulos activos", func(t *testing.T) {
		p := profile(t, domain.RoleCashier)
		p.Modules = []string{"variants"}
		p.Permissions = []domain.Permission{domain.PermCatalogRead, domain.PermInventoryRead, domain.PermSalesCreate, domain.PermSalesRead}
		var gotTenant, gotUser uuid.UUID
		uc := &stubUseCases{me: func(tenantID, userID uuid.UUID) (app.Profile, error) {
			gotTenant, gotUser = tenantID, userID
			return p, nil
		}}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		principal := auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "cashier"}
		req = req.WithContext(auth.WithPrincipal(req.Context(), principal))

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.Tenant.ID(), gotTenant)
		require.Equal(t, p.User.ID(), gotUser)
		require.JSONEq(t, `{
			"id": "`+p.User.ID().String()+`",
			"tenant_id": "`+p.Tenant.ID().String()+`",
			"tenant_name": "Ferretería El Tornillo",
			"tenant_nit": "900123456-8",
			"business_template": "ferreteria",
			"default_tax_rate": "0.19",
			"name": "Ana Gómez",
			"email": "ana@eltornillo.co",
			"role": "cashier",
			"permissions": ["catalog:read", "inventory:read", "sales:create", "sales:read"],
			"modules": ["variants"]
		}`, rec.Body.String())
	})

	t.Run("sin módulos activos envía listas vacías, no null", func(t *testing.T) {
		p := profile(t, domain.RoleCashier)
		uc := &stubUseCases{me: func(uuid.UUID, uuid.UUID) (app.Profile, error) { return p, nil }}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "cashier"}))

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"modules":[]`)
		require.Contains(t, rec.Body.String(), `"permissions":[]`)
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		uc := &stubUseCases{}
		requireError(t, serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)), http.StatusUnauthorized, "unauthorized")
	})

	t.Run("un usuario que ya no existe o está inactivo responde 401", func(t *testing.T) {
		uc := &stubUseCases{me: func(uuid.UUID, uuid.UUID) (app.Profile, error) { return app.Profile{}, domain.ErrUserNotFound }}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "cashier"}))

		requireError(t, serve(t, uc, req), http.StatusUnauthorized, "unauthorized")
	})
}

func TestUpdateTenant(t *testing.T) {
	patch := func(p app.Profile, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenant", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		principal := auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "owner"}
		return req.WithContext(auth.WithPrincipal(req.Context(), principal))
	}

	t.Run("pasa el IVA del token y del cuerpo al caso de uso y responde la empresa", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		var gotTenant, gotUser uuid.UUID
		var got app.TenantUpdate
		uc := &stubUseCases{update: func(tenantID, userID uuid.UUID, in app.TenantUpdate) (*domain.Tenant, error) {
			gotTenant, gotUser, got = tenantID, userID, in
			require.NoError(t, p.Tenant.SetDefaultTaxRate(*in.DefaultTaxRate))
			return p.Tenant, nil
		}}

		rec := serve(t, uc, patch(p, `{"default_tax_rate":"0.05"}`))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.Tenant.ID(), gotTenant)
		require.Equal(t, p.User.ID(), gotUser)
		require.Equal(t, "0.05", got.DefaultTaxRate.String())
		require.JSONEq(t, `{
			"id": "`+p.Tenant.ID().String()+`",
			"name": "Ferretería El Tornillo",
			"nit": "900123456-8",
			"business_template": "ferreteria",
			"default_tax_rate": "0.05"
		}`, rec.Body.String())
	})

	t.Run("un IVA inválido responde 422 invalid_default_tax_rate", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		uc := &stubUseCases{update: func(uuid.UUID, uuid.UUID, app.TenantUpdate) (*domain.Tenant, error) {
			return nil, domain.ErrInvalidDefaultTaxRate
		}}

		requireError(t, serve(t, uc, patch(p, `{"default_tax_rate":"0.12345"}`)), http.StatusUnprocessableEntity, "invalid_default_tax_rate")
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenant", strings.NewReader(`{"default_tax_rate":"0.19"}`))
		req.Header.Set("Content-Type", "application/json")
		requireError(t, serve(t, &stubUseCases{}, req), http.StatusUnauthorized, "unauthorized")
	})
}
