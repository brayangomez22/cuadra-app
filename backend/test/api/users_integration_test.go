//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserManagement runs the user management through the whole stack:
// invitations, roles, deactivation, password resets and their audit.
func TestUserManagement(t *testing.T) {
	f := newSalesFlow(t)
	do, owner := f.do, f.ownerToken
	me := func(t *testing.T, token string) (status int, id string) {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/me", "", token)
		var body struct{ ID string }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body.ID
	}
	_, ownerID := me(t, owner)

	t.Run("solo owner y admin gestionan usuarios", func(t *testing.T) {
		admin := joinStaff(t, do, owner, "admin")
		require.Equal(t, http.StatusOK, do(t, http.MethodGet, "/api/v1/users", "", admin.Token).Code)
		for _, token := range []string{f.cashierToken, f.warehouseToken} {
			require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/users", "", token).Code)
			rec := do(t, http.MethodPost, "/api/v1/invitations", `{"email":"x@equipo.co","name":"X","role":"cashier"}`, token)
			require.Equal(t, http.StatusForbidden, rec.Code)
		}
	})

	t.Run("un usuario desactivado pierde el acceso en su siguiente request", func(t *testing.T) {
		cashier := joinStaff(t, do, owner, "cashier")
		status, _ := me(t, cashier.Token)
		require.Equal(t, http.StatusOK, status)

		rec := do(t, http.MethodPost, "/api/v1/users/"+cashier.ID+"/deactivate", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		status, _ = me(t, cashier.Token)
		require.Equal(t, http.StatusUnauthorized, status, "the access token still lives, but the user does not")

		rec = do(t, http.MethodPost, "/api/v1/users/"+cashier.ID+"/activate", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		status, _ = me(t, cashier.Token)
		require.Equal(t, http.StatusOK, status)
	})

	t.Run("el rol cambiado aplica sin volver a iniciar sesión", func(t *testing.T) {
		member := joinStaff(t, do, owner, "warehouse")
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/users", "", member.Token).Code)
		rec := do(t, http.MethodPatch, "/api/v1/users/"+member.ID, `{"role":"admin"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, http.StatusOK, do(t, http.MethodGet, "/api/v1/users", "", member.Token).Code)
	})

	t.Run("el dueño no puede quitarse su propio rol de owner si es el único", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/users/"+ownerID, `{"role":"admin"}`, owner)
		requireErrorCode(t, rec, http.StatusConflict, "last_owner")
		rec = do(t, http.MethodPost, "/api/v1/users/"+ownerID+"/deactivate", "", owner)
		requireErrorCode(t, rec, http.StatusConflict, "self_deactivation")
	})

	t.Run("un admin no nombra dueños ni toca al dueño", func(t *testing.T) {
		admin := joinStaff(t, do, owner, "admin")
		cashier := joinStaff(t, do, owner, "cashier")
		rec := do(t, http.MethodPatch, "/api/v1/users/"+cashier.ID, `{"role":"owner"}`, admin.Token)
		requireErrorCode(t, rec, http.StatusForbidden, "owner_only")
		rec = do(t, http.MethodPost, "/api/v1/users/"+ownerID+"/deactivate", "", admin.Token)
		requireErrorCode(t, rec, http.StatusForbidden, "owner_only")
	})

	t.Run("el enlace de restablecimiento cambia la contraseña una vez y queda en la auditoría", func(t *testing.T) {
		cashier := joinStaff(t, do, owner, "cashier")
		rec := do(t, http.MethodPost, "/api/v1/users/"+cashier.ID+"/password-reset", "", owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var link struct{ Token string }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &link))

		body := `{"token":"` + link.Token + `","password":"nueva-clave-segura-7"}`
		require.Equal(t, http.StatusNoContent, do(t, http.MethodPost, "/api/v1/auth/password-reset", body, "").Code)
		requireErrorCode(t, do(t, http.MethodPost, "/api/v1/auth/password-reset", body, ""), http.StatusGone, "link_used")
		status, _ := me(t, cashier.Token)
		require.Equal(t, http.StatusOK, status, "the access token lives until it expires; the refresh tokens were revoked")

		rec = do(t, http.MethodGet, "/api/v1/audit-events?action=user.password_reset", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), cashier.ID)
		require.NotContains(t, rec.Body.String(), link.Token)
	})

	t.Run("los usuarios de otra empresa no existen", func(t *testing.T) {
		other, _ := f.signup(t, "otra@empresa.co")
		rec := do(t, http.MethodPost, "/api/v1/users/"+ownerID+"/deactivate", "", other)
		requireErrorCode(t, rec, http.StatusNotFound, "not_found")
	})
}

// requireErrorCode checks the status and the error code of a response.
func requireErrorCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	require.Equal(t, status, rec.Code, rec.Body.String())
	require.Equal(t, code, body.Error.Code)
}
