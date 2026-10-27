package httpx_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestLimitBody(t *testing.T) {
	read := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := httpx.LimitBody(10)(read)

	t.Run("deja pasar un cuerpo dentro del límite", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789")))
		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("responde 413 a un cuerpo declarado más grande, sin leerlo", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789A")))
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		require.Contains(t, rec.Body.String(), "request_too_large")
	})

	t.Run("corta la lectura de un cuerpo sin largo declarado", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", io.MultiReader(strings.NewReader("0123456789A")))
		req.ContentLength = -1
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
