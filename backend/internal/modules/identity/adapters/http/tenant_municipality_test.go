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

func TestUpdateTenantMunicipality(t *testing.T) {
	patch := func(p app.Profile, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenant", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		principal := auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "owner"}
		return req.WithContext(auth.WithPrincipal(req.Context(), principal))
	}

	t.Run("pasa el municipio al caso de uso y lo responde en la empresa y en /me", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		var got app.TenantUpdate
		uc := &stubUseCases{update: func(_, _ uuid.UUID, in app.TenantUpdate) (*domain.Tenant, error) {
			got = in
			contact, err := p.Tenant.Contact().WithMunicipality(*in.MunicipalityCode)
			require.NoError(t, err)
			p.Tenant.SetContact(contact)
			return p.Tenant, nil
		}}

		rec := serve(t, uc, patch(p, `{"municipality_code":"05001"}`))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "05001", *got.MunicipalityCode)
		require.Contains(t, rec.Body.String(), `"municipality_code":"05001"`)
	})

	t.Run("un municipio que no existe responde 422 invalid_municipality", func(t *testing.T) {
		p := profile(t, domain.RoleOwner)
		uc := &stubUseCases{update: func(uuid.UUID, uuid.UUID, app.TenantUpdate) (*domain.Tenant, error) {
			return nil, domain.ErrInvalidMunicipality
		}}

		requireError(t, serve(t, uc, patch(p, `{"municipality_code":"99999"}`)), http.StatusUnprocessableEntity, "invalid_municipality")
	})

}
