package httpadapter_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestSignUp(t *testing.T) {
	body := `{"tenant_name":"Ferretería El Tornillo","nit":"900123456-8","name":"Ana Gómez","email":"ana@eltornillo.co","password":"clave-segura-123"}`

	t.Run("responde 201 con la sesión y la cookie de refresh segura", func(t *testing.T) {
		want := session(t)
		var got app.SignUpInput
		uc := &stubUseCases{signUp: func(in app.SignUpInput) (app.Session, error) { got = in; return want, nil }}

		rec := serve(t, uc, post("/api/v1/auth/signup", body))

		requireSession(t, rec, http.StatusCreated, want)
		require.Equal(t, app.SignUpInput{
			TenantName: "Ferretería El Tornillo", NIT: "900123456-8", OwnerName: "Ana Gómez",
			Email: "ana@eltornillo.co", Password: "clave-segura-123",
		}, got, "without business_template the use case picks the default")
	})

	t.Run("pasa la plantilla de tipo de negocio elegida", func(t *testing.T) {
		var got app.SignUpInput
		uc := &stubUseCases{signUp: func(in app.SignUpInput) (app.Session, error) { got = in; return session(t), nil }}
		withTemplate := strings.Replace(body, `{`, `{"business_template":"ropa_calzado",`, 1)

		rec := serve(t, uc, post("/api/v1/auth/signup", withTemplate))

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "ropa_calzado", got.BusinessTemplate)
	})

	t.Run("traduce los errores de validación del dominio a 422 con su código", func(t *testing.T) {
		tests := []struct {
			err  error
			code string
		}{
			{err: domain.ErrInvalidEmail, code: "invalid_email"},
			{err: domain.ErrInvalidNIT, code: "invalid_nit"},
			{err: domain.ErrInvalidNITCheckDigit, code: "invalid_nit_check_digit"},
			{err: domain.ErrPasswordTooShort, code: "password_too_short"},
			{err: domain.ErrPasswordTooLong, code: "password_too_long"},
			{err: domain.ErrInvalidTenantName, code: "invalid_tenant_name"},
			{err: domain.ErrInvalidUserName, code: "invalid_user_name"},
			{err: domain.ErrUnknownBusinessTemplate, code: "unknown_business_template"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				uc := &stubUseCases{signUp: func(app.SignUpInput) (app.Session, error) { return app.Session{}, tt.err }}
				requireError(t, serve(t, uc, post("/api/v1/auth/signup", body)), http.StatusUnprocessableEntity, tt.code)
			})
		}
	})

	t.Run("un error inesperado responde 500 sin detalles", func(t *testing.T) {
		uc := &stubUseCases{signUp: func(app.SignUpInput) (app.Session, error) {
			return app.Session{}, errors.New("db: connection refused at 10.0.0.5")
		}}

		rec := serve(t, uc, post("/api/v1/auth/signup", body))

		requireError(t, rec, http.StatusInternalServerError, "internal_error")
		require.NotContains(t, rec.Body.String(), "10.0.0.5")
	})
}

func TestListBusinessTemplates(t *testing.T) {
	t.Run("lista las plantillas con su nombre y descripción, sin autenticación", func(t *testing.T) {
		uc := &stubUseCases{templates: []app.BusinessTemplate{
			{ID: "ferreteria", Name: "Ferretería", Description: "Herramientas y construcción."},
			{ID: "ropa_calzado", Name: "Ropa y calzado", Description: "Prendas por talla y color."},
		}}

		rec := serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/business-templates", nil))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items": [
			{"id": "ferreteria", "name": "Ferretería", "description": "Herramientas y construcción."},
			{"id": "ropa_calzado", "name": "Ropa y calzado", "description": "Prendas por talla y color."}
		]}`, rec.Body.String())
	})
}
