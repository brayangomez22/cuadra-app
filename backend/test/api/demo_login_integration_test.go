//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// demoSession is what POST /auth/demo and the login answer.
type demoSession struct {
	AccessToken string `json:"access_token"`
	User        struct {
		TenantID         uuid.UUID `json:"tenant_id"`
		Role             string
		BusinessTemplate string `json:"business_template"`
		Demo             bool
	} `json:"user"`
}

func decodeSession(t *testing.T, body []byte) demoSession {
	t.Helper()
	var s demoSession
	require.NoError(t, json.Unmarshal(body, &s))
	return s
}

// TestDemoLogin enters the public demo without a password: only demo
// tenants, only with the demo on.
func TestDemoLogin(t *testing.T) {
	e := newDemoEnv(t)
	on, off := e.publicAPI(t, true), e.publicAPI(t, false)
	tenants := map[string]uuid.UUID{}
	for _, template := range []string{"ferreteria", "ropa_calzado"} {
		id, err := e.resetter.Reset(t.Context(), template)
		require.NoError(t, err)
		tenants[template] = id
	}

	cases := []struct{ template, role string }{
		{"ferreteria", "cashier"}, {"ferreteria", "owner"}, {"ropa_calzado", "warehouse"}, {"ropa_calzado", "admin"},
	}
	for _, c := range cases {
		t.Run("entra con el rol elegido al tenant demo: "+c.template+" "+c.role, func(t *testing.T) {
			rec := call(t, on, http.MethodPost, "/api/v1/auth/demo", `{"business_template":"`+c.template+`","role":"`+c.role+`"}`, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Header().Get("Set-Cookie"), "cuadra_refresh=")
			s := decodeSession(t, rec.Body.Bytes())
			require.Equal(t, tenants[c.template], s.User.TenantID)
			require.Equal(t, c.role, s.User.Role)
			require.True(t, s.User.Demo)

			rec = call(t, on, http.MethodGet, "/api/v1/me", "", s.AccessToken)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"demo":true`)
		})
	}

	t.Run("lista los tipos de negocio con demo", func(t *testing.T) {
		rec := call(t, on, http.MethodGet, "/api/v1/auth/demo", "", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list struct {
			Items []struct {
				BusinessTemplate string `json:"business_template"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		var got []string
		for _, it := range list.Items {
			got = append(got, it.BusinessTemplate)
		}
		require.ElementsMatch(t, []string{"ferreteria", "ropa_calzado"}, got)
	})

	t.Run("con la demo apagada responde 404 y no abre sesión", func(t *testing.T) {
		rec := call(t, off, http.MethodPost, "/api/v1/auth/demo", `{"business_template":"ferreteria","role":"owner"}`, "")
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "demo_unavailable")
		require.Empty(t, rec.Header().Get("Set-Cookie"))
		require.Equal(t, http.StatusNotFound, call(t, off, http.MethodGet, "/api/v1/auth/demo", "", "").Code)
	})

	t.Run("nunca abre sesión en un tenant que no es demo", func(t *testing.T) {
		rec := call(t, on, http.MethodPost, "/api/v1/auth/signup", `{"tenant_name":"Droguería Real","nit":"900123456-8",
			"name":"Ana","email":"ana@drogueria-real.test","password":"clave-segura-123","business_template":"drogueria"}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.False(t, decodeSession(t, rec.Body.Bytes()).User.Demo)

		rec = call(t, on, http.MethodPost, "/api/v1/auth/demo", `{"business_template":"drogueria","role":"owner"}`, "")
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "demo_unavailable")
	})

	t.Run("un rol que no existe responde 400", func(t *testing.T) {
		rec := call(t, on, http.MethodPost, "/api/v1/auth/demo", `{"business_template":"ferreteria","role":"root"}`, "")
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}
