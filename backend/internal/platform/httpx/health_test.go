package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		status  string
	}{
		{"healthz responde 200 con JSON status ok", httpx.Healthz, "ok"},
		{"readyz responde 200 con JSON status ready", httpx.Readyz, "ready"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			tt.handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, map[string]string{"status": tt.status}, body)
		})
	}
}
