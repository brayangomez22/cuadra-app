package modules_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// fakeStore keeps the active modules of each tenant in memory. Like the
// PostgreSQL store, Update runs one tenant's changes one at a time.
type fakeStore struct {
	mu      sync.Mutex
	active  map[uuid.UUID][]string
	changes []modules.Change
}

func newFakeStore() *fakeStore { return &fakeStore{active: map[uuid.UUID][]string{}} }

func (f *fakeStore) Active(_ context.Context, tenantID uuid.UUID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.active[tenantID]), nil
}

func (f *fakeStore) Update(_ context.Context, tenantID uuid.UUID, decide func(active []string) (*modules.Change, error)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	change, err := decide(slices.Clone(f.active[tenantID]))
	if err != nil || change == nil {
		return err
	}
	f.changes = append(f.changes, *change)
	if change.Active {
		f.active[tenantID] = append(f.active[tenantID], change.Module)
	} else {
		f.active[tenantID] = slices.DeleteFunc(f.active[tenantID], func(id string) bool { return id == change.Module })
	}
	return nil
}

type env struct {
	svc     *modules.Service
	store   *fakeStore
	now     time.Time
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

func newEnv(t *testing.T) *env {
	t.Helper()
	registry, err := modules.NewRegistry(testDefinitions()...)
	require.NoError(t, err)
	e := &env{
		store:   newFakeStore(),
		now:     time.Date(2026, 10, 16, 15, 0, 0, 0, time.UTC),
		spans:   tracetest.NewSpanRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	e.svc, err = modules.NewService(modules.Deps{
		Registry:       registry,
		Store:          e.store,
		Now:            func() time.Time { return e.now },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	return e
}

// changes returns cuadra.modules.changes for a module and an action.
func (e *env) changes(t *testing.T, module, action string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "cuadra.modules.changes" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			for _, dp := range sum.DataPoints {
				id, _ := dp.Attributes.Value("module.id")
				act, _ := dp.Attributes.Value("action")
				if id.AsString() == module && act.AsString() == action {
					total += dp.Value
				}
			}
		}
	}
	return total
}

// span returns the only ended span named name.
func (e *env) span(t *testing.T, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	return found[0]
}

func TestSetActive(t *testing.T) {
	tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	t.Run("activa un módulo cuando sus dependencias están activas", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)

		status, err := e.svc.SetActive(t.Context(), tenantID, userID, "lots_expiry", true)

		require.NoError(t, err)
		require.True(t, status.Active)
		require.Equal(t, "lots_expiry", status.ID)
		active, err := e.svc.Active(t.Context(), tenantID)
		require.NoError(t, err)
		require.Equal(t, []string{"variants", "lots_expiry"}, active)
		require.Equal(t, modules.Change{Module: "lots_expiry", Active: true, By: userID, At: e.now}, e.store.changes[1])

		require.EqualValues(t, 1, e.changes(t, "lots_expiry", "enabled"))
		span := e.spans.Ended()[len(e.spans.Ended())-1]
		require.Equal(t, "modules.EnableModule", span.Name())
		require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenantID.String()))
		require.Contains(t, span.Attributes(), attribute.String("user.id", userID.String()))
		require.Contains(t, span.Attributes(), attribute.String("module.id", "lots_expiry"))
	})

	t.Run("activar un módulo con una dependencia inactiva falla con un error claro", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "lots_expiry", true)

		require.ErrorIs(t, err, modules.ErrDependencyInactive)
		var depErr *modules.DependencyError
		require.ErrorAs(t, err, &depErr)
		require.Equal(t, "variants", depErr.Module)
		require.Empty(t, e.store.changes)
		span := e.span(t, "modules.EnableModule")
		require.Equal(t, codes.Error, span.Status().Code)
	})

	t.Run("activar un módulo ya activo no falla ni cuenta un cambio", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "work_orders", true)
		require.NoError(t, err)

		status, err := e.svc.SetActive(t.Context(), tenantID, userID, "work_orders", true)

		require.NoError(t, err)
		require.True(t, status.Active)
		require.Len(t, e.store.changes, 1)
		require.EqualValues(t, 1, e.changes(t, "work_orders", "enabled"))
	})

	t.Run("desactivar un módulo del que depende otro activo falla", func(t *testing.T) {
		e := newEnv(t)
		for _, id := range []string{"variants", "lots_expiry"} {
			_, err := e.svc.SetActive(t.Context(), tenantID, userID, id, true)
			require.NoError(t, err)
		}

		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", false)

		require.ErrorIs(t, err, modules.ErrRequiredBy)
		var depErr *modules.DependencyError
		require.ErrorAs(t, err, &depErr)
		require.Equal(t, "lots_expiry", depErr.Module)
		active, err := e.svc.Active(t.Context(), tenantID)
		require.NoError(t, err)
		require.Equal(t, []string{"variants", "lots_expiry"}, active)
	})

	t.Run("desactiva un módulo sin dependientes activos", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)

		status, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", false)

		require.NoError(t, err)
		require.False(t, status.Active)
		active, err := e.svc.IsActive(t.Context(), tenantID, "variants")
		require.NoError(t, err)
		require.False(t, active)
		require.EqualValues(t, 1, e.changes(t, "variants", "disabled"))
		e.span(t, "modules.DisableModule")
	})

	t.Run("desactivar un módulo no borra datos: solo cambia su estado", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)

		_, err = e.svc.SetActive(t.Context(), tenantID, userID, "variants", false)

		require.NoError(t, err)
		require.Equal(t, []modules.Change{
			{Module: "variants", Active: true, By: userID, At: e.now},
			{Module: "variants", Active: false, By: userID, At: e.now},
		}, e.store.changes)
	})

	t.Run("un módulo desconocido responde ErrUnknownModule", func(t *testing.T) {
		e := newEnv(t)
		for _, active := range []bool{true, false} {
			_, err := e.svc.SetActive(t.Context(), tenantID, userID, "price_lists", active)
			require.ErrorIs(t, err, modules.ErrUnknownModule)
		}
		require.Empty(t, e.store.changes)
	})

	t.Run("los módulos de un tenant no afectan a otro", func(t *testing.T) {
		e := newEnv(t)
		other := uuid.Must(uuid.NewV7())
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)

		_, err = e.svc.SetActive(t.Context(), other, userID, "lots_expiry", true)

		require.ErrorIs(t, err, modules.ErrDependencyInactive)
	})
}

func TestActive(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())

	t.Run("ignora los módulos guardados que ya no están registrados y sigue el orden del registro", func(t *testing.T) {
		e := newEnv(t)
		e.store.active[tenantID] = []string{"work_orders", "retired_module", "variants"}

		active, err := e.svc.Active(t.Context(), tenantID)

		require.NoError(t, err)
		require.Equal(t, []string{"variants", "work_orders"}, active)
		ok, err := e.svc.IsActive(t.Context(), tenantID, "retired_module")
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("un tenant sin módulos no tiene ninguno activo", func(t *testing.T) {
		e := newEnv(t)
		active, err := e.svc.Active(t.Context(), tenantID)
		require.NoError(t, err)
		require.Empty(t, active)
	})
}

func TestList(t *testing.T) {
	t.Run("lista todos los módulos registrados con su estado y sus dependencias", func(t *testing.T) {
		e := newEnv(t)
		tenantID := uuid.Must(uuid.NewV7())
		e.store.active[tenantID] = []string{"variants"}

		list, err := e.svc.List(t.Context(), tenantID)

		require.NoError(t, err)
		require.Len(t, list, 3)
		require.Equal(t, "variants", list[0].ID)
		require.True(t, list[0].Active)
		require.Equal(t, "lots_expiry", list[1].ID)
		require.False(t, list[1].Active)
		require.Equal(t, []string{"variants"}, list[1].DependsOn)
		e.span(t, "modules.ListModules")
	})
}

func TestNewService(t *testing.T) {
	t.Run("falla sin registro o sin store", func(t *testing.T) {
		_, err := modules.NewService(modules.Deps{Store: newFakeStore()})
		require.Error(t, err)
		registry, err := modules.NewRegistry()
		require.NoError(t, err)
		_, err = modules.NewService(modules.Deps{Registry: registry})
		require.Error(t, err)
	})

	t.Run("un error del store se propaga", func(t *testing.T) {
		registry, err := modules.NewRegistry(testDefinitions()...)
		require.NoError(t, err)
		boom := errors.New("boom")
		svc, err := modules.NewService(modules.Deps{Registry: registry, Store: failingStore{boom}})
		require.NoError(t, err)

		_, err = svc.SetActive(t.Context(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), "variants", true)
		require.ErrorIs(t, err, boom)
		_, err = svc.Active(t.Context(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, boom)
	})
}

type failingStore struct{ err error }

func (f failingStore) Active(context.Context, uuid.UUID) ([]string, error) { return nil, f.err }
func (f failingStore) Update(context.Context, uuid.UUID, func([]string) (*modules.Change, error)) error {
	return f.err
}

func TestSetUpTenant(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())

	t.Run("preparar un tenant activa los módulos de su plantilla sin usuario", func(t *testing.T) {
		e := newEnv(t)

		require.NoError(t, e.svc.SetUpTenant(t.Context(), tenantID, []string{"lots_expiry", "variants"}))

		active, err := e.svc.Active(t.Context(), tenantID)
		require.NoError(t, err)
		require.Equal(t, []string{"variants", "lots_expiry"}, active)
		require.Len(t, e.store.changes, 2)
		for _, c := range e.store.changes {
			require.Equal(t, uuid.Nil, c.By, "no user turned it on")
			require.Equal(t, e.now, c.At)
			require.True(t, c.Active)
		}
		require.Zero(t, e.changes(t, "variants", "enabled"), "seeding a new tenant is not a change of its modules")
		span := e.span(t, "modules.SetUpTenant")
		require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenantID.String()))
	})

	t.Run("sin módulos no activa ninguno", func(t *testing.T) {
		e := newEnv(t)

		require.NoError(t, e.svc.SetUpTenant(t.Context(), tenantID, nil))

		require.Empty(t, e.store.changes)
	})

	t.Run("rechaza un módulo no registrado sin activar ninguno", func(t *testing.T) {
		e := newEnv(t)

		err := e.svc.SetUpTenant(t.Context(), tenantID, []string{"variants", "nope"})

		require.ErrorIs(t, err, modules.ErrUnknownModule)
		require.Empty(t, e.store.changes)
		require.Equal(t, codes.Error, e.span(t, "modules.SetUpTenant").Status().Code)
	})

	t.Run("rechaza un módulo sin su dependencia sin activar ninguno", func(t *testing.T) {
		e := newEnv(t)

		err := e.svc.SetUpTenant(t.Context(), tenantID, []string{"work_orders", "lots_expiry"})

		require.ErrorIs(t, err, modules.ErrDependencyInactive)
		var depErr *modules.DependencyError
		require.ErrorAs(t, err, &depErr)
		require.Equal(t, "variants", depErr.Module)
		require.Empty(t, e.store.changes)
	})
}
