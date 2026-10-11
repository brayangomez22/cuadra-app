package errreport_test

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/errreport"
)

const (
	jwt          = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIwMTkyIn0.Sfl-KxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	refreshToken = "0192f0a1-1234-7abc-8def-0123456789ab.q3J0Zm9vYmFyYmF6cXV4cXV1eGNvcmdlZ3JhdWx0YWJj"
)

func TestScrub(t *testing.T) {
	tests := []struct {
		name string
		// event is the event before scrubbing.
		event *sentry.Event
		// gone must not appear anywhere in the scrubbed event.
		gone []string
		// check asserts on the fields that must survive.
		check func(t *testing.T, e *sentry.Event)
	}{
		{
			name: "elimina emails del mensaje y de la excepción",
			event: &sentry.Event{
				Message: "usuario ana.perez@ferreteria.co no encontrado",
				Exception: []sentry.Exception{{
					Type:  "*pgconn.PgError",
					Value: `duplicate key value violates unique constraint "users_email_key": Key (email)=(ana+pos@gmail.com) already exists`,
				}},
			},
			gone: []string{"ana.perez@ferreteria.co", "ana+pos@gmail.com"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, "usuario [email] no encontrado", e.Message)
				require.Contains(t, e.Exception[0].Value, "users_email_key")
			},
		},
		{
			name: "elimina tokens JWT y Bearer",
			event: &sentry.Event{
				Message:   "token rechazado: " + jwt,
				Exception: []sentry.Exception{{Value: "upstream answered 401 for Authorization: Bearer abc123.def-456_ghi"}},
			},
			gone: []string{jwt, "abc123.def-456_ghi"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, "token rechazado: [token]", e.Message)
				require.Contains(t, e.Exception[0].Value, "Bearer [token]")
			},
		},
		{
			name:  "elimina refresh tokens con formato tenant.secreto",
			event: &sentry.Event{Message: "refresh token " + refreshToken + " reused"},
			gone:  []string{refreshToken, "q3J0Zm9vYmFyYmF6"},
		},
		{
			name: "elimina contraseñas escritas en el texto y en cadenas de conexión",
			event: &sentry.Event{
				Message:   "login failed password=hunter2 secret: s3cr3t-value",
				Exception: []sentry.Exception{{Value: "failed to connect to postgres://app_user:Sup3rS3cret@db:5432/cuadra"}},
			},
			gone: []string{"hunter2", "s3cr3t-value", "Sup3rS3cret", "app_user:"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Contains(t, e.Exception[0].Value, "postgres://[filtered]@db:5432/cuadra")
			},
		},
		{
			name: "reemplaza valores de claves sensibles en contexts",
			event: &sentry.Event{
				Contexts: map[string]sentry.Context{
					"app": {
						"api_key":       "k-123456",
						"name":          "cuadra-api",
						"Authorization": "Basic dXNlcjpwYXNz",
						"customerEmail": "cliente@correo.com",
					},
					"job": {
						"password": "hunter2",
						"nested":   map[string]any{"refresh_token": "opaque-value", "items": 3},
						"count":    3,
					},
				},
			},
			gone: []string{"hunter2", "dXNlcjpwYXNz", "opaque-value", "cliente@correo.com", "k-123456"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, "cuadra-api", e.Contexts["app"]["name"])
				require.Equal(t, 3, e.Contexts["job"]["count"])
				require.Equal(t, 3, e.Contexts["job"]["nested"].(map[string]any)["items"])
				require.Equal(t, "[filtered]", e.Contexts["job"]["password"])
			},
		},
		{
			name: "elimina body, cookies, headers sensibles, query string y env de la request",
			event: &sentry.Event{
				Request: &sentry.Request{
					URL:         "https://api.cuadra.co/api/v1/auth/login?email=ana@correo.com",
					Method:      "POST",
					Data:        `{"email":"ana@correo.com","password":"hunter2"}`,
					QueryString: "email=ana@correo.com",
					Cookies:     "cuadra_refresh=" + refreshToken,
					Headers: map[string]string{
						"Authorization": "Bearer " + jwt,
						"Cookie":        "cuadra_refresh=" + refreshToken,
						"X-Api-Key":     "k-123456",
						"User-Agent":    "Mozilla/5.0",
						"Content-Type":  "application/json",
					},
					Env: map[string]string{"REMOTE_ADDR": "181.50.10.20"},
				},
			},
			gone: []string{"ana@correo.com", "hunter2", refreshToken, jwt, "k-123456", "181.50.10.20"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, "https://api.cuadra.co/api/v1/auth/login", e.Request.URL)
				require.Equal(t, "POST", e.Request.Method)
				require.Empty(t, e.Request.Data)
				require.Empty(t, e.Request.QueryString)
				require.Empty(t, e.Request.Cookies)
				require.Empty(t, e.Request.Env)
				require.Equal(t, map[string]string{"User-Agent": "Mozilla/5.0", "Content-Type": "application/json"}, e.Request.Headers)
			},
		},
		{
			name: "deja user.id pero borra email, username, nombre, IP y datos extra",
			event: &sentry.Event{
				User: sentry.User{
					ID:        "0192f0a1-1234-7abc-8def-0123456789ab",
					Email:     "ana@correo.com",
					Username:  "ana",
					Name:      "Ana Pérez",
					IPAddress: "181.50.10.20",
					Data:      map[string]string{"phone": "3001234567"},
				},
			},
			gone: []string{"ana@correo.com", "Ana Pérez", "181.50.10.20", "3001234567"},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, sentry.User{ID: "0192f0a1-1234-7abc-8def-0123456789ab"}, e.User)
			},
		},
		{
			name: "filtra emails y tokens de los breadcrumbs",
			event: &sentry.Event{
				Breadcrumbs: []*sentry.Breadcrumb{{
					Message: "login de ana@correo.com",
					Data:    map[string]any{"token": "opaque-value", "url": "/api/v1/me?access=" + jwt, "status": 401},
				}},
			},
			gone: []string{"ana@correo.com", "opaque-value", jwt},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, 401, e.Breadcrumbs[0].Data["status"])
			},
		},
		{
			name: "no modifica un evento sin datos sensibles",
			event: &sentry.Event{
				Message:   "stock insuficiente del producto 0192f0a1-1234-7abc-8def-0123456789ab",
				Exception: []sentry.Exception{{Type: "*errors.errorString", Value: "inventory: insufficient stock"}},
				Tags:      map[string]string{"tenant_id": "0192f0a1-0000-7000-8000-0000000000aa", "request_id": "4bf92f3577b34da6"},
				Contexts: map[string]sentry.Context{
					"trace": {"trace_id": "4bf92f3577b34da6a3ce929d0e0e4736", "span_id": "00f067aa0ba902b7"},
				},
			},
			check: func(t *testing.T, e *sentry.Event) {
				require.Equal(t, "stock insuficiente del producto 0192f0a1-1234-7abc-8def-0123456789ab", e.Message)
				require.Equal(t, "inventory: insufficient stock", e.Exception[0].Value)
				require.Equal(t, "0192f0a1-0000-7000-8000-0000000000aa", e.Tags["tenant_id"])
				require.Equal(t, "4bf92f3577b34da6", e.Tags["request_id"])
				require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", e.Contexts["trace"]["trace_id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errreport.Scrub(tt.event, nil)
			require.NotNil(t, got)

			raw, err := json.Marshal(got)
			require.NoError(t, err)
			for _, s := range tt.gone {
				require.NotContains(t, string(raw), s)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
