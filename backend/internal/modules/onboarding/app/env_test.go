package app_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
)

var now = time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)

// The steps of two templates: a hardware store and a clothing store.
var templateSteps = map[string][]string{
	"ferreteria":   {"business_profile", "invoicing", "products", "alternate_units", "initial_stock", "customer_credit", "invite_team", "first_sale"},
	"ropa_calzado": {"business_profile", "invoicing", "products", "variant_products", "initial_stock", "invite_team", "first_sale"},
}

type env struct {
	svc        *app.Service
	state      *fakeState
	dismissals *fakeDismissals
	invoicing  *fakeInvoicing
	modules    []string
	perms      []string
	spans      *tracetest.SpanRecorder
	metrics    *sdkmetric.ManualReader
	actor      app.Actor
}

func newEnv(t *testing.T, template string) *env {
	t.Helper()
	e := &env{
		state:      &fakeState{template: template},
		dismissals: newFakeDismissals(),
		invoicing:  &fakeInvoicing{},
		modules:    []string{"unit_conversions", "variants"},
		spans:      tracetest.NewSpanRecorder(),
		metrics:    sdkmetric.NewManualReader(),
		actor:      app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"},
	}
	svc, err := app.NewService(app.Deps{
		State:      e.state,
		Dismissals: e.dismissals,
		Templates:  fakeTemplates{},
		Modules:    e,
		Invoicing:  e.invoicing,
		Can: func(role, permission string) bool {
			return role == "owner" || slices.Contains(e.perms, permission)
		},
		Now:            func() time.Time { return now },
		TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans)),
		MeterProvider:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics)),
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// Active is the tenant's active modules (the env is its own modules port).
func (e *env) Active(context.Context, uuid.UUID) ([]string, error) { return e.modules, nil }

func (e *env) span(t *testing.T, name string) {
	t.Helper()
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			return
		}
	}
	t.Fatalf("no span %q", name)
}

// counted sums the data points of an Int64 counter.
func (e *env) counted(t *testing.T, name string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(context.Background(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
					total += dp.Value
				}
			}
		}
	}
	return total
}
