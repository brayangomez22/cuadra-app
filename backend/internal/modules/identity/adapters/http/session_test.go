package httpadapter_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestRefreshSession(t *testing.T) {
	withCookie := func(value string) *http.Request {
		req := post("/api/v1/auth/refresh", "")
		req.AddCookie(&http.Cookie{Name: "cuadra_refresh", Value: value})
		return req
	}

	t.Run("rota la cookie y devuelve un access token nuevo", func(t *testing.T) {
		want := session(t)
		var got string
		uc := &stubUseCases{refresh: func(raw string) (app.Session, error) { got = raw; return want, nil }}

		rec := serve(t, uc, withCookie("old-refresh-token"))

		requireSession(t, rec, http.StatusOK, want)
		require.Equal(t, "old-refresh-token", got)
	})

	t.Run("sin cookie responde 401 invalid_session y borra la cookie", func(t *testing.T) {
		uc := &stubUseCases{refresh: func(string) (app.Session, error) {
			t.Fatal("must not be called without a cookie")
			return app.Session{}, nil
		}}

		rec := serve(t, uc, post("/api/v1/auth/refresh", ""))

		requireError(t, rec, http.StatusUnauthorized, "invalid_session")
		requireClearedCookie(t, rec)
	})

	t.Run("un token inválido, vencido o reutilizado responde 401 invalid_session y borra la cookie", func(t *testing.T) {
		for _, cause := range []error{domain.ErrInvalidRefreshToken, domain.ErrRefreshTokenExpired, domain.ErrRefreshTokenReused} {
			uc := &stubUseCases{refresh: func(string) (app.Session, error) { return app.Session{}, cause }}

			rec := serve(t, uc, withCookie("x"))

			requireError(t, rec, http.StatusUnauthorized, "invalid_session")
			requireClearedCookie(t, rec)
		}
	})

	t.Run("un error inesperado responde 500 y conserva la cookie", func(t *testing.T) {
		uc := &stubUseCases{refresh: func(string) (app.Session, error) { return app.Session{}, errors.New("db down") }}

		rec := serve(t, uc, withCookie("x"))

		requireError(t, rec, http.StatusInternalServerError, "internal_error")
		require.Empty(t, rec.Header().Values("Set-Cookie"))
	})
}

func TestLogout(t *testing.T) {
	t.Run("responde 204, revoca el token de la cookie y la borra", func(t *testing.T) {
		var got string
		uc := &stubUseCases{logout: func(raw string) error { got = raw; return nil }}
		req := post("/api/v1/auth/logout", "")
		req.AddCookie(&http.Cookie{Name: "cuadra_refresh", Value: "refresh-token"})

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "refresh-token", got)
		requireClearedCookie(t, rec)
	})

	t.Run("sin cookie también responde 204", func(t *testing.T) {
		uc := &stubUseCases{logout: func(raw string) error { require.Empty(t, raw); return nil }}

		rec := serve(t, uc, post("/api/v1/auth/logout", ""))

		require.Equal(t, http.StatusNoContent, rec.Code)
		requireClearedCookie(t, rec)
	})
}
