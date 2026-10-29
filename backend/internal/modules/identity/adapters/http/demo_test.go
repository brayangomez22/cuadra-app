package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestStartDemo(t *testing.T) {
	body := `{"business_template":"ropa_calzado","role":"cashier"}`

	t.Run("responde 200 con la sesión y su cookie, y pasa la IP sin el puerto", func(t *testing.T) {
		want := session(t)
		var got app.DemoLoginInput
		uc := &stubUseCases{demo: func(in app.DemoLoginInput) (app.Session, error) { got = in; return want, nil }}
		req := post("/api/v1/auth/demo", body)
		req.RemoteAddr = "203.0.113.7:51234"

		rec := serve(t, uc, req)

		requireSession(t, rec, http.StatusOK, want)
		require.Equal(t, app.DemoLoginInput{BusinessTemplate: "ropa_calzado", Role: "cashier", ClientIP: "203.0.113.7"}, got)
	})

	t.Run("sin demo responde 404 demo_unavailable", func(t *testing.T) {
		uc := &stubUseCases{demo: func(app.DemoLoginInput) (app.Session, error) { return app.Session{}, domain.ErrDemoUnavailable }}
		requireError(t, serve(t, uc, post("/api/v1/auth/demo", body)), http.StatusNotFound, "demo_unavailable")
	})

	t.Run("demasiados intentos responden 429 con Retry-After", func(t *testing.T) {
		uc := &stubUseCases{demo: func(app.DemoLoginInput) (app.Session, error) {
			return app.Session{}, &app.RateLimitError{RetryAfter: 30 * time.Second}
		}}
		rec := serve(t, uc, post("/api/v1/auth/demo", body))
		requireError(t, rec, http.StatusTooManyRequests, "too_many_requests")
		require.Equal(t, "30", rec.Header().Get("Retry-After"))
	})
}

func TestListDemos(t *testing.T) {
	t.Run("lista los tipos de negocio con demo", func(t *testing.T) {
		uc := &stubUseCases{demos: func() ([]app.BusinessTemplate, error) {
			return []app.BusinessTemplate{{ID: "ferreteria", Name: "Ferretería"}, {ID: "ropa_calzado", Name: "Ropa y calzado"}}, nil
		}}
		rec := serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/auth/demo", nil))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got struct {
			Items []struct {
				BusinessTemplate string `json:"business_template"`
				Name             string `json:"name"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Len(t, got.Items, 2)
		require.Equal(t, "ferreteria", got.Items[0].BusinessTemplate)
		require.Equal(t, "Ropa y calzado", got.Items[1].Name)
	})

	t.Run("con la demo apagada responde 404 demo_unavailable", func(t *testing.T) {
		uc := &stubUseCases{demos: func() ([]app.BusinessTemplate, error) { return nil, domain.ErrDemoUnavailable }}
		requireError(t, serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/auth/demo", nil)), http.StatusNotFound, "demo_unavailable")
	})
}
