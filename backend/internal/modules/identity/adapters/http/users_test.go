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

// as returns req authenticated as p's user.
func as(p app.Profile, req *http.Request) *http.Request {
	principal := auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: p.User.Role().String()}
	return req.WithContext(auth.WithPrincipal(req.Context(), principal))
}

func TestUserHandlers(t *testing.T) {
	owner := profile(t, domain.RoleOwner)
	target := uuid.Must(uuid.NewV7())
	patch := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+target.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return as(owner, req)
	}

	t.Run("pasa el actor del token y responde el usuario cambiado", func(t *testing.T) {
		var got app.Actor
		uc := &stubUseCases{stubUserUseCases: stubUserUseCases{changeUser: func(a app.Actor, _ uuid.UUID) (*domain.User, error) {
			got = a
			return owner.User, nil
		}}}
		rec := serve(t, uc, patch(`{"role":"admin"}`))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.Actor{TenantID: owner.Tenant.ID(), UserID: owner.User.ID()}, got)
		require.Contains(t, rec.Body.String(), `"active":true`)
	})

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"un usuario de otra empresa responde 404", domain.ErrUserNotFound, http.StatusNotFound, "not_found"},
		{"una acción sobre un dueño sin serlo responde 403", domain.ErrOwnerOnly, http.StatusForbidden, "owner_only"},
		{"dejar la empresa sin dueño responde 409", domain.ErrLastOwner, http.StatusConflict, "last_owner"},
		{"desactivarse a sí mismo responde 409", domain.ErrSelfDeactivation, http.StatusConflict, "self_deactivation"},
		{"un rol desconocido responde 422", domain.ErrInvalidRole, http.StatusUnprocessableEntity, "invalid_role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCases{stubUserUseCases: stubUserUseCases{changeUser: func(app.Actor, uuid.UUID) (*domain.User, error) {
				return nil, tt.err
			}}}
			requireError(t, serve(t, uc, patch(`{"role":"admin"}`)), tt.status, tt.code)
		})
	}
}

func TestLinkHandlers(t *testing.T) {
	t.Run("aceptar la invitación abre la sesión con la cookie", func(t *testing.T) {
		want := session(t)
		var got app.AcceptInvitationInput
		uc := &stubUseCases{stubUserUseCases: stubUserUseCases{accept: func(in app.AcceptInvitationInput) (app.Session, error) {
			got = in
			return want, nil
		}}}
		rec := serve(t, uc, post("/api/v1/auth/invitations/accept", `{"token":"t.x","name":"Luisa","password":"clave-segura-123"}`))
		requireSession(t, rec, http.StatusOK, want)
		require.Equal(t, "t.x", got.Token)
	})

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"un enlace inventado responde 404", domain.ErrLinkNotFound, http.StatusNotFound, "link_not_found"},
		{"un enlace vencido responde 410", domain.ErrLinkExpired, http.StatusGone, "link_expired"},
		{"un enlace usado responde 410", domain.ErrLinkUsed, http.StatusGone, "link_used"},
		{"una contraseña común responde 422", domain.ErrPasswordTooCommon, http.StatusUnprocessableEntity, "password_too_common"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCases{stubUserUseCases: stubUserUseCases{reset: func(app.ResetPasswordInput) error { return tt.err }}}
			rec := serve(t, uc, post("/api/v1/auth/password-reset", `{"token":"t.x","password":"clave-segura-123"}`))
			requireError(t, rec, tt.status, tt.code)
		})
	}

	t.Run("la contraseña actual equivocada responde 422", func(t *testing.T) {
		uc := &stubUseCases{stubUserUseCases: stubUserUseCases{changeOwnPw: func(app.Actor, app.PasswordChange) error {
			return domain.ErrWrongPassword
		}}}
		req := as(profile(t, domain.RoleCashier), post("/api/v1/me/password", `{"current_password":"x","new_password":"clave-segura-123"}`))
		requireError(t, serve(t, uc, req), http.StatusUnprocessableEntity, "wrong_password")
	})
}
