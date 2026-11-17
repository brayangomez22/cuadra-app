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

func TestRejectNUL(t *testing.T) {
	var got string
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(http.StatusNoContent)
	})
	handler := httpx.RejectNUL(echo)
	serve := func(target, contentType, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	t.Run("rechaza un texto con NUL en el JSON, que la base no puede guardar", func(t *testing.T) {
		rec := serve("/api/v1/units", "application/json", `{"code":"caja\u0000","name":"Caja"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "invalid_request")
	})

	t.Run("rechaza un NUL en una clave anidada o en una lista", func(t *testing.T) {
		rec := serve("/api/v1/products", "application/json", `{"attributes":{"marca":["a","b\u0000"]}}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rechaza un NUL en la query", func(t *testing.T) {
		rec := serve("/api/v1/suppliers?q=%27%00x", "", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("deja pasar intacto un JSON sin NUL, aunque escriba la secuencia como texto", func(t *testing.T) {
		body := `{"name":"Ruta C:\\u0000 de prueba","price":"12500.00"}`
		rec := serve("/api/v1/units", "application/json", body)
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, body, got)
	})

	t.Run("no revisa un archivo subido: un .xlsx es binario", func(t *testing.T) {
		rec := serve("/api/v1/imports", "multipart/form-data; boundary=x", "PK\x03\x04\x00\x00")
		require.Equal(t, http.StatusNoContent, rec.Code)
	})
}
