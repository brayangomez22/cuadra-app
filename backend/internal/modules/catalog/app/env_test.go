package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc   *app.Service
	store *store
	actor app.Actor
	// meters is a unit of the actor's tenant.
	meters  *domain.UnitOfMeasure
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

var testNow = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store: &store{
			products:   map[uuid.UUID]*domain.Product{},
			categories: map[uuid.UUID]*domain.Category{},
			units:      map[uuid.UUID]*domain.UnitOfMeasure{},
			attributes: map[uuid.UUID]*domain.AttributeDefinition{},
			axes:       map[uuid.UUID]*domain.VariantAxis{},
		},
		actor: app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())},
		spans: tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Products:       fakeProducts{e.store},
		Categories:     fakeCategories{e.store},
		Units:          fakeUnits{e.store},
		Attributes:     fakeAttributes{e.store},
		VariantAxes:    fakeAxes{e.store},
		Seeds:          fakeSeeds{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	e.meters = e.addUnit(t, e.actor.TenantID, "m", "Metro")
	// The axes the variant tests use, as a tenant would define them.
	e.addAxis(t, e.actor.TenantID, "talla", "S", "M", "L")
	e.addAxis(t, e.actor.TenantID, "color", "Rojo", "Azul")
	e.addAxis(t, e.actor.TenantID, "presentación", "Galón", "Cuarto")
	return e
}

// addAxis stores a variant axis of tenantID directly in the fake.
func (e *env) addAxis(t *testing.T, tenantID uuid.UUID, name string, values ...string) *domain.VariantAxis {
	t.Helper()
	a, err := domain.NewVariantAxis(tenantID, name, values, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeAxes{e.store}.Create(t.Context(), a))
	return a
}

// addAttribute stores an attribute definition of tenantID directly in the fake.
func (e *env) addAttribute(t *testing.T, tenantID uuid.UUID, p domain.AttributeParams) *domain.AttributeDefinition {
	t.Helper()
	d, err := domain.NewAttributeDefinition(tenantID, p, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeAttributes{e.store}.Create(t.Context(), d))
	return d
}

// addUnit stores a unit of tenantID directly in the fake, without the
// service.
func (e *env) addUnit(t *testing.T, tenantID uuid.UUID, code, name string) *domain.UnitOfMeasure {
	t.Helper()
	u, err := domain.NewUnit(tenantID, code, name, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeUnits{e.store}.Create(t.Context(), u))
	return u
}

// span returns the attributes of the only ended span named name.
func (e *env) span(t *testing.T, name string) (sdktrace.ReadOnlySpan, map[attribute.Key]attribute.Value) {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range found[0].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return found[0], attrs
}

// counter returns the sum of the data points of an int64 counter whose
// attribute key equals value (any data point when key is "").
func (e *env) counter(t *testing.T, name, key, value string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is not an int64 counter", name)
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value(attribute.Key(key)); key == "" || ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}
