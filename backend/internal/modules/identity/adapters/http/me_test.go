package httpadapter_test

import (
	"net/http"
	"net/http/httptest"
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
		contact, err := domain.NewContact("Calle 45 # 12-30", "Medellín", "604 444 1234")
		require.NoError(t, err)
		p.Tenant.SetContact(contact)
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
			"tenant_address": "Calle 45 # 12-30",
			"tenant_city": "Medellín",
			"tenant_phone": "604 444 1234",
			"business_template": "ferreteria",
			"default_tax_rate": "0.19",
			"name": "Ana Gómez",
			"email": "ana@eltornillo.co",
			"role": "cashier",
			"permissions": ["catalog:read", "inventory:read", "sales:create", "sales:read"],
			"modules": ["variants"],
			"demo": false
		}`, rec.Body.String())
	})

	t.Run("dice demo en un tenant de demostración", func(t *testing.T) {
		p := profile(t, domain.RoleWarehouse)
		p.Tenant.MarkDemo()
		uc := &stubUseCases{me: func(uuid.UUID, uuid.UUID) (app.Profile, error) { return p, nil }}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "warehouse"}))

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"demo":true`)
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
