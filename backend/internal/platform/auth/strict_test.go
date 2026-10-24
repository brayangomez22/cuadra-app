package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// strictHandler has the shape of the StrictHandlerFunc of each package
// oapi-codegen generates, which is a distinct type in each one.
type strictHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error)

func TestRequirePrincipal(t *testing.T) {
	principal := auth.Principal{TenantID: uuid.New(), UserID: uuid.New(), Role: "cashier"}

	t.Run("deja pasar una petición con principal", func(t *testing.T) {
		called := false
		next := strictHandler(func(context.Context, http.ResponseWriter, *http.Request, any) (any, error) {
			called = true
			return "ok", nil
		})
		ctx := auth.WithPrincipal(context.Background(), principal)

		got, err := auth.RequirePrincipal(next, "ListProducts")(ctx, httptest.NewRecorder(), nil, nil)

		require.NoError(t, err)
		require.Equal(t, "ok", got)
		require.True(t, called)
	})

	t.Run("responde 401 sin llamar al caso de uso cuando no llega un principal", func(t *testing.T) {
		called := false
		next := strictHandler(func(context.Context, http.ResponseWriter, *http.Request, any) (any, error) {
			called = true
			return "no debía llegar", nil
		})
		w := httptest.NewRecorder()

		got, err := auth.RequirePrincipal(next, "ListProducts")(context.Background(), w, nil, nil)

		require.NoError(t, err)
		require.Nil(t, got)
		require.False(t, called)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "Bearer", w.Header().Get("WWW-Authenticate"))
		var body struct {
			Error struct{ Code string } `json:"error"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "unauthorized", body.Error.Code)
	})
}

func TestSeesCosts(t *testing.T) {
	cases := []struct {
		name string
		role string
		want bool
	}{
		{"quien edita el catálogo ve costos y márgenes", "admin", true},
		{"quien no lo edita no los ve", "cashier", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := auth.WithPrincipal(context.Background(), auth.Principal{Role: c.role})

			require.Equal(t, c.want, auth.SeesCosts(ctx, can))
		})
	}

	t.Run("sin principal no ve costos", func(t *testing.T) {
		require.False(t, auth.SeesCosts(context.Background(), func(string, string) bool { return true }))
	})
}
