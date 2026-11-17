package flags_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
)

type serviceEnv struct {
	svc      *flags.Service
	eval     *flags.Evaluator
	store    *fakeStore
	operator uuid.UUID
	spans    *tracetest.SpanRecorder
	metrics  *sdkmetric.ManualReader
}

func newServiceEnv(t *testing.T) *serviceEnv {
	t.Helper()
	e := &serviceEnv{
		store:    newFakeStore(),
		operator: uuid.Must(uuid.NewV7()),
		spans:    tracetest.NewSpanRecorder(),
		metrics:  sdkmetric.NewManualReader(),
	}
	e.eval = newEvaluator(t, e.store, newClock())
	svc, err := flags.NewService(flags.ServiceDeps{
		Evaluator:      e.eval,
		Operators:      []uuid.UUID{e.operator},
		Now:            newClock().Now,
		TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans)),
		MeterProvider:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics)),
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

func (e *serviceEnv) spanNames() []string {
	var out []string
	for _, s := range e.spans.Ended() {
		out = append(out, s.Name())
	}
	return out
}

func (e *serviceEnv) changes(t *testing.T) map[attribute.Distinct]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(context.Background(), &rm))
	out := map[attribute.Distinct]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "cuadra.flags.changes" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				out[dp.Attributes.Equivalent()] = dp.Value
			}
		}
	}
	return out
}

func TestServiceOperators(t *testing.T) {
	e := newServiceEnv(t)
	tenant := e.store.addTenant("Ferretería La Tuerca")
	stranger := uuid.Must(uuid.NewV7())

	t.Run("reconoce solo a los operadores configurados", func(t *testing.T) {
		require.True(t, e.svc.IsOperator(e.operator))
		require.False(t, e.svc.IsOperator(stranger))
	})

	t.Run("quien no es operador no ve ni cambia las flags", func(t *testing.T) {
		_, err := e.svc.List(t.Context(), stranger)
		require.ErrorIs(t, err, flags.ErrNotOperator)
		_, err = e.svc.SetGlobal(t.Context(), stranger, "new_pos", true)
		require.ErrorIs(t, err, flags.ErrNotOperator)
		require.ErrorIs(t, e.svc.SetException(t.Context(), stranger, tenant, "new_pos", true), flags.ErrNotOperator)
		require.ErrorIs(t, e.svc.ClearException(t.Context(), stranger, tenant, "new_pos"), flags.ErrNotOperator)
	})
}

func TestServiceChanges(t *testing.T) {
	t.Run("prender una flag para todos se ve de inmediato en esta instancia", func(t *testing.T) {
		e := newServiceEnv(t)
		hardware := e.store.addTenant("Ferretería La Tuerca")
		clothing := e.store.addTenant("Calzado Andino")
		require.False(t, e.eval.Enabled(t.Context(), hardware, "new_pos"))

		status, err := e.svc.SetGlobal(t.Context(), e.operator, "new_pos", true)
		require.NoError(t, err)
		require.True(t, status.Global)

		require.True(t, e.eval.Enabled(t.Context(), hardware, "new_pos"))
		require.True(t, e.eval.Enabled(t.Context(), clothing, "new_pos"))
		require.Contains(t, e.spanNames(), "flags.SetGlobal")
		want := attribute.NewSet(
			attribute.String("flag", "new_pos"), attribute.String("scope", "global"), attribute.String("action", "enabled"))
		require.Equal(t, int64(1), e.changes(t)[want.Equivalent()])
	})

	t.Run("una excepción solo cambia la flag del tenant y se puede quitar", func(t *testing.T) {
		e := newServiceEnv(t)
		pilot := e.store.addTenant("Ferretería La Tuerca")
		other := e.store.addTenant("Calzado Andino")

		require.NoError(t, e.svc.SetException(t.Context(), e.operator, pilot, "new_pos", true))
		require.True(t, e.eval.Enabled(t.Context(), pilot, "new_pos"))
		require.False(t, e.eval.Enabled(t.Context(), other, "new_pos"))

		require.NoError(t, e.svc.ClearException(t.Context(), e.operator, pilot, "new_pos"))
		require.False(t, e.eval.Enabled(t.Context(), pilot, "new_pos"))
		require.Subset(t, e.spanNames(), []string{"flags.SetException", "flags.ClearException"})
	})

	t.Run("rechaza una flag que el código no declara", func(t *testing.T) {
		e := newServiceEnv(t)
		tenant := e.store.addTenant("Droguería Central")

		_, err := e.svc.SetGlobal(t.Context(), e.operator, "retired", true)
		require.ErrorIs(t, err, flags.ErrUnknownFlag)
		require.ErrorIs(t, e.svc.SetException(t.Context(), e.operator, tenant, "retired", true), flags.ErrUnknownFlag)
	})

	t.Run("rechaza la excepción de un tenant que no existe", func(t *testing.T) {
		e := newServiceEnv(t)

		err := e.svc.SetException(t.Context(), e.operator, uuid.Must(uuid.NewV7()), "new_pos", true)
		require.ErrorIs(t, err, flags.ErrUnknownTenant)
	})
}

func TestServiceList(t *testing.T) {
	t.Run("lista las flags declaradas con su valor global y sus excepciones", func(t *testing.T) {
		e := newServiceEnv(t)
		pilot := e.store.addTenant("Ferretería La Tuerca")
		e.store.setGlobal("quick_sale", true)
		e.store.setException(pilot, "new_pos", true)
		e.store.setGlobal("retired", true)

		got, err := e.svc.List(t.Context(), e.operator)
		require.NoError(t, err)

		require.Len(t, got, 2)
		require.Equal(t, "new_pos", got[0].Key)
		require.False(t, got[0].Global)
		require.Equal(t, []flags.Exception{{TenantID: pilot, TenantName: "Ferretería La Tuerca", Key: "new_pos", Enabled: true}}, got[0].Exceptions)
		require.Equal(t, "quick_sale", got[1].Key)
		require.True(t, got[1].Global)
		require.Empty(t, got[1].Exceptions)
		require.Contains(t, e.spanNames(), "flags.ListFlags")
	})
}
