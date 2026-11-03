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
			"default_tax_rate": "0.05",
			"address": "",
			"city": "",
			"phone": ""
		}`, rec.Body.String())
	})

	t.Run("pasa la dirección, la ciudad y el teléfono al caso de uso y responde los guardados", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		var got app.TenantUpdate
		uc := &stubUseCases{update: func(_, _ uuid.UUID, in app.TenantUpdate) (*domain.Tenant, error) {
			got = in
			contact, err := domain.NewContact(*in.Address, *in.City, "")
			require.NoError(t, err)
			p.Tenant.SetContact(contact)
			return p.Tenant, nil
		}}

		rec := serve(t, uc, patch(p, `{"address":" Cra. 7 # 20-15 ","city":"Bogotá D.C."}`))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, " Cra. 7 # 20-15 ", *got.Address)
		require.Equal(t, "Bogotá D.C.", *got.City)
		require.Nil(t, got.Phone)
		require.Nil(t, got.DefaultTaxRate)
		require.Contains(t, rec.Body.String(), `"address":"Cra. 7 # 20-15"`)
	})

	t.Run("un teléfono inválido responde 422 invalid_tenant_phone", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		uc := &stubUseCases{update: func(uuid.UUID, uuid.UUID, app.TenantUpdate) (*domain.Tenant, error) {
			return nil, domain.ErrInvalidTenantPhone
		}}

		requireError(t, serve(t, uc, patch(p, `{"phone":"llamar"}`)), http.StatusUnprocessableEntity, "invalid_tenant_phone")
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenant", strings.NewReader(`{"default_tax_rate":"0.19"}`))
		req.Header.Set("Content-Type", "application/json")
		requireError(t, serve(t, &stubUseCases{}, req), http.StatusUnauthorized, "unauthorized")
	})
}
