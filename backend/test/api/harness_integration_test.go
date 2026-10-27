//go:build integration

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/server"
)

// testSecret signs the access tokens of the API under test.
var testSecret = []byte("0123456789abcdef0123456789abcdef")

// newAPI builds the whole API on a database of its own, as cmd/api does
// (internal/server), without telemetry. tokens issues tokens for any role.
func newAPI(t *testing.T) (handler http.Handler, tokens *auth.JWT) {
	t.Helper()
	handler, tokens, _ = newAPIWithJobs(t)
	return handler, tokens
}

// newAPIWithJobs is newAPI with the client of its background jobs, which
// works them once started.
func newAPIWithJobs(t *testing.T) (handler http.Handler, tokens *auth.JWT, jobClient *jobs.Client) {
	t.Helper()
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	database := dbtest.New(t)
	workers := river.NewWorkers()
	jobClient, err = jobs.New(jobs.Config{DB: database, Workers: workers})
	require.NoError(t, err)
	handler, err = server.New(server.Config{
		DB:             database,
		Tokens:         tokens,
		Jobs:           jobClient,
		Workers:        workers,
		Log:            slog.New(slog.DiscardHandler),
		TracerProvider: noop.NewTracerProvider(),
		MeterProvider:  metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	return handler, tokens, jobClient
}

// requestFunc sends a request to the API under test with a bearer token.
type requestFunc func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder

// unitID returns the id of the tenant's unit with code, as GET /units lists
// it: products name their unit by id.
func unitID(t *testing.T, do requestFunc, bearer, code string) string {
	t.Helper()
	rec := do(t, http.MethodGet, "/api/v1/units", "", bearer)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var units struct {
		Items []struct{ ID, Code string } `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &units))
	for _, u := range units.Items {
		if u.Code == code {
			return u.ID
		}
	}
	t.Fatalf("the tenant has no unit %q", code)
	return ""
}
