package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestClientIP(t *testing.T) {
	ingress := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/16")}
	cases := []struct {
		name, remote, forwarded string
		trusted                 []netip.Prefix
		want                    string
	}{
		{"sin proxies de confianza usa la IP de la conexión e ignora X-Forwarded-For",
			"203.0.113.7:51000", "198.51.100.9", nil, "203.0.113.7"},
		{"detrás del ingress toma la IP que el ingress agregó",
			"10.0.3.4:443", "198.51.100.9", ingress, "198.51.100.9"},
		{"un cliente no falsifica su IP anteponiendo otra",
			"10.0.3.4:443", "1.2.3.4, 198.51.100.9", ingress, "198.51.100.9"},
		{"salta los proxies de confianza encadenados",
			"10.0.3.4:443", "198.51.100.9, 10.0.8.1", ingress, "198.51.100.9"},
		{"una conexión que no viene del ingress no puede usar X-Forwarded-For",
			"203.0.113.7:51000", "198.51.100.9", ingress, "203.0.113.7"},
		{"un valor inválido no reemplaza la IP de la conexión",
			"10.0.3.4:443", "no-es-ip", ingress, "10.0.3.4"},
		{"detrás del ingress sin X-Forwarded-For usa la IP de la conexión",
			"10.0.3.4:443", "", ingress, "10.0.3.4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := httpx.ClientIP(tc.trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = httpx.ClientIPFrom(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("sin el middleware no hay IP", func(t *testing.T) {
		require.Empty(t, httpx.ClientIPFrom(t.Context()))
	})
}
