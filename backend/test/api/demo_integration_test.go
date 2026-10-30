//go:build integration

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/server"
)

const demoPassword = "demo-clave-segura"

// demoToday is the real day of the tests' resets: the history ends the day
// before.
var demoToday = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

// demoEnv is a resetter of small demos (a few weeks, a short catalog) on a
// database of its own.
type demoEnv struct {
	resetter *demo.Resetter
	db       *db.DB
	jobs     *jobs.Client
}

func newDemoEnv(t *testing.T) demoEnv {
	t.Helper()
	database := dbtest.New(t)
	jobClient, err := jobs.New(jobs.Config{DB: database})
	require.NoError(t, err)
	r, err := demo.NewResetter(demo.Resetter{
		Generator: demo.Generator{DB: database, Jobs: jobClient, Log: slog.New(slog.DiscardHandler)},
		Templates: []string{"ferreteria", "ropa_calzado"},
		Password:  demoPassword,
		Options:   demo.Options{Seed: 3, Days: 21, Products: 60, SalesPerDay: 12, Customers: 20},
		Now:       func() time.Time { return demoToday },
	})
	require.NoError(t, err)
	return demoEnv{resetter: r, db: database, jobs: jobClient}
}

// newDemo builds a resetter of small demos and the API that serves it to
// its users, with the demo on.
func newDemo(t *testing.T) (*demo.Resetter, http.Handler) {
	t.Helper()
	e := newDemoEnv(t)
	return e.resetter, e.publicAPI(t, true)
}

// publicAPI is the API as cmd/api builds it, on the real clock, with the
// demo on or off (DEMO_ENABLED).
func (e demoEnv) publicAPI(t *testing.T, demoEnabled bool) http.Handler {
	t.Helper()
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	h, err := server.New(server.Config{
		DB: e.db, Tokens: tokens, Jobs: e.jobs, Workers: river.NewWorkers(), Log: slog.New(slog.DiscardHandler),
		DemoEnabled: demoEnabled,
	})
	require.NoError(t, err)
	return h
}

// call sends a JSON request to h and returns the recorder.
func call(t *testing.T, h http.Handler, method, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// logInDemo logs in a demo user through the public API.
func logInDemo(t *testing.T, h http.Handler, email string, tenant uuid.UUID) string {
	t.Helper()
	rec := call(t, h, http.MethodPost, "/api/v1/auth/login",
		`{"email":"`+email+`","password":"`+demoPassword+`","tenant_id":"`+tenant.String()+`"}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	return session.AccessToken
}

func TestDemoReset(t *testing.T) {
	r, h := newDemo(t)

	for _, template := range []string{"ferreteria", "ropa_calzado"} {
		t.Run("generar dos veces con la misma semilla deja la misma huella: "+template, func(t *testing.T) {
			first, err := r.Reset(t.Context(), template)
			require.NoError(t, err)
			want, err := r.Fingerprint(t.Context(), first, template)
			require.NoError(t, err)
			require.Positive(t, want.Sales)
			require.Positive(t, want.Purchases)
			require.Positive(t, want.Alerts, "P111: the demo shows reorder alerts")
			require.Positive(t, want.Counts, "P135: the demo shows physical counts with their accuracy")

			second, err := r.Reset(t.Context(), template)
			require.NoError(t, err)
			require.NotEqual(t, first, second)
			got, err := r.Fingerprint(t.Context(), second, template)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}

	t.Run("el reinicio deja el tenant con la huella inicial después de cambiarlo", func(t *testing.T) {
		tenant, err := r.Reset(t.Context(), "ferreteria")
		require.NoError(t, err)
		want, err := r.Fingerprint(t.Context(), tenant, "ferreteria")
		require.NoError(t, err)

		owner := logInDemo(t, h, "dueno@ferreteria.demo.cuadra.co", tenant)
		do := func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
			t.Helper()
			return call(t, h, method, path, body, bearer)
		}
		rec := call(t, h, http.MethodPost, "/api/v1/products", `{"name":"Artículo de prueba","sku":"PRUEBA-1",
			"base_unit_id":"`+unitID(t, do, owner, "und")+`","cost":"100.00","price":"200.00","tax_rate":"0.19","attributes":{}}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		changed, err := r.Fingerprint(t.Context(), tenant, "ferreteria")
		require.NoError(t, err)
		require.NotEqual(t, want, changed)

		again, err := r.Reset(t.Context(), "ferreteria")
		require.NoError(t, err)
		got, err := r.Fingerprint(t.Context(), again, "ferreteria")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("el demo tiene un usuario por rol con la contraseña de la demo", func(t *testing.T) {
		tenant, err := r.Reset(t.Context(), "ropa_calzado")
		require.NoError(t, err)
		for _, user := range []string{"dueno", "admin", "cajero", "bodega"} {
			require.NotEmpty(t, logInDemo(t, h, user+"@ropa.demo.cuadra.co", tenant))
		}
	})

	t.Run("aislamiento: el reinicio no toca a otra empresa", func(t *testing.T) {
		rec := call(t, h, http.MethodPost, "/api/v1/auth/signup", `{"tenant_name":"Ferretería Real","nit":"900123456-8",
			"name":"Ana","email":"ana@real.test","password":"clave-segura-123","business_template":"ferreteria"}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		_, err := r.Reset(t.Context(), "ferreteria")
		require.NoError(t, err)

		rec = call(t, h, http.MethodPost, "/api/v1/auth/login", `{"email":"ana@real.test","password":"clave-segura-123"}`, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
}
