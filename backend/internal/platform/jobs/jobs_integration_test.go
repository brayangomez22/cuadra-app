//go:build integration

package jobs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// waitTimeout bounds how long a test waits for River to work a job.
const waitTimeout = 30 * time.Second

type okArgs struct {
	Marker uuid.UUID `json:"marker"`
}

func (okArgs) Kind() string { return "test_ok" }

type okWorker struct {
	river.WorkerDefaults[okArgs]
}

func (okWorker) Work(context.Context, *river.Job[okArgs]) error { return nil }

type failArgs struct {
	Marker uuid.UUID `json:"marker"`
}

func (failArgs) Kind() string { return "test_fail" }

type failWorker struct {
	river.WorkerDefaults[failArgs]
}

func (failWorker) Work(context.Context, *river.Job[failArgs]) error {
	return errors.New("el proveedor no responde")
}

type periodicArgs struct{}

func (periodicArgs) Kind() string { return "test_periodic" }

type periodicWorker struct {
	river.WorkerDefaults[periodicArgs]
	ran chan struct{}
}

func (w periodicWorker) Work(context.Context, *river.Job[periodicArgs]) error {
	select {
	case w.ran <- struct{}{}:
	default:
	}
	return nil
}

// immediateRetry retries a failed job right away so the test does not wait
// for River's exponential backoff.
type immediateRetry struct{}

func (immediateRetry) NextRetry(*rivertype.JobRow) time.Time { return time.Now() }

type harness struct {
	db      *db.DB
	client  *jobs.Client
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
	tracer  trace.Tracer
	ran     chan struct{}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		db:      dbtest.New(t),
		spans:   tracetest.NewSpanRecorder(),
		metrics: sdkmetric.NewManualReader(),
		ran:     make(chan struct{}, 1),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(h.spans))
	h.tracer = tp.Tracer("test")

	workers := river.NewWorkers()
	river.AddWorker(workers, okWorker{})
	river.AddWorker(workers, failWorker{})
	river.AddWorker(workers, periodicWorker{ran: h.ran})

	client, err := jobs.New(jobs.Config{
		DB:      h.db,
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(river.PeriodicInterval(time.Hour), func() (river.JobArgs, *river.InsertOpts) {
				return periodicArgs{}, nil
			}, &river.PeriodicJobOpts{RunOnStart: true}),
		},
		RetryPolicy:    immediateRetry{},
		TracerProvider: tp,
		MeterProvider:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(h.metrics)),
	})
	require.NoError(t, err)
	require.NoError(t, client.Start(t.Context()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, client.Stop(ctx))
	})
	h.client = client
	return h
}

// enqueue inserts args in a committed transaction and returns the job id.
func (h *harness) enqueue(ctx context.Context, t *testing.T, args river.JobArgs, opts *river.InsertOpts) int64 {
	t.Helper()
	var id int64
	require.NoError(t, h.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		id, err = h.client.InsertTx(ctx, tx, args, opts)
		return err
	}))
	return id
}

type jobState struct {
	State    string
	Attempt  int
	ErrCount int
}

func (h *harness) job(t *testing.T, id int64) (jobState, bool) {
	t.Helper()
	var s jobState
	err := h.db.WithTx(t.Context(), func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(),
			"SELECT state::text, attempt, coalesce(array_length(errors, 1), 0) FROM river_job WHERE id = $1", id,
		).Scan(&s.State, &s.Attempt, &s.ErrCount)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return s, false
	}
	require.NoError(t, err)
	return s, true
}

func (h *harness) waitState(t *testing.T, id int64, state string) jobState {
	t.Helper()
	var last jobState
	require.Eventually(t, func() bool {
		s, ok := h.job(t, id)
		last = s
		return ok && s.State == state
	}, waitTimeout, 50*time.Millisecond, "el trabajo %d no llegó a %q (último estado %+v)", id, state, last)
	return last
}

func (h *harness) spanFor(t *testing.T, name string, jobID int64) sdktrace.ReadOnlySpan {
	t.Helper()
	var found sdktrace.ReadOnlySpan
	require.Eventually(t, func() bool {
		for _, s := range h.spans.Ended() {
			if s.Name() != name {
				continue
			}
			for _, a := range s.Attributes() {
				if a.Key == "job.id" && a.Value.AsInt64() == jobID {
					found = s
					return true
				}
			}
		}
		return false
	}, waitTimeout, 50*time.Millisecond, "no se encontró el span %q del trabajo %d", name, jobID)
	return found
}

func (h *harness) collect(t *testing.T) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, h.metrics.Collect(t.Context(), &rm))
	out := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			out[m.Name] = m
		}
	}
	return out
}

func sumFor(m metricdata.Metrics, kind string, extra ...func(metricdata.DataPoint[int64]) bool) int64 {
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		return 0
	}
	var total int64
	for _, dp := range sum.DataPoints {
		if v, ok := dp.Attributes.Value("job.kind"); !ok || v.AsString() != kind {
			continue
		}
		match := true
		for _, f := range extra {
			match = match && f(dp)
		}
		if match {
			total += dp.Value
		}
	}
	return total
}

func TestJobs(t *testing.T) {
	h := newHarness(t)

	t.Run("si la transacción hace rollback, el trabajo no existe", func(t *testing.T) {
		errRollback := errors.New("falló el caso de uso")
		var id int64
		err := h.db.WithTx(t.Context(), func(tx pgx.Tx) error {
			var err error
			id, err = h.client.InsertTx(t.Context(), tx, okArgs{Marker: uuid.New()}, nil)
			require.NoError(t, err)
			return errRollback
		})
		require.ErrorIs(t, err, errRollback)
		require.NotZero(t, id)

		_, exists := h.job(t, id)
		require.False(t, exists)
	})

	t.Run("si la transacción hace commit, el trabajo se encola y se ejecuta", func(t *testing.T) {
		id := h.enqueue(t.Context(), t, okArgs{Marker: uuid.New()}, nil)

		s := h.waitState(t, id, "completed")
		require.Equal(t, 1, s.Attempt)
	})

	t.Run("un trabajo que falla se reintenta y queda descartado tras agotar los intentos", func(t *testing.T) {
		id := h.enqueue(t.Context(), t, failArgs{Marker: uuid.New()}, &river.InsertOpts{MaxAttempts: 3})

		s := h.waitState(t, id, "discarded")
		require.Equal(t, 3, s.Attempt)
		require.Equal(t, 3, s.ErrCount)

		span := h.spanFor(t, "jobs.test_fail", id)
		require.Equal(t, "Error", span.Status().Code.String())
		require.NotEmpty(t, span.Events(), "el error queda registrado en el span")
	})

	t.Run("la traza del worker queda enlazada a la del request que lo originó", func(t *testing.T) {
		ctx, request := h.tracer.Start(t.Context(), "POST /api/v1/sales")
		id := h.enqueue(ctx, t, okArgs{Marker: uuid.New()}, nil)
		request.End()

		span := h.spanFor(t, "jobs.test_ok", id)
		require.NotEqual(t, request.SpanContext().TraceID(), span.SpanContext().TraceID(),
			"un reintento no debe alargar la traza del request")
		require.Len(t, span.Links(), 1)
		require.Equal(t, request.SpanContext().TraceID(), span.Links()[0].SpanContext.TraceID())
		require.Equal(t, request.SpanContext().SpanID(), span.Links()[0].SpanContext.SpanID())
	})

	t.Run("un trabajo periódico se ejecuta al arrancar", func(t *testing.T) {
		select {
		case <-h.ran:
		case <-time.After(waitTimeout):
			t.Fatal("el trabajo periódico no se ejecutó")
		}
	})

	t.Run("registra trabajos encolados, completados, fallidos y la latencia de la cola", func(t *testing.T) {
		before := h.collect(t)
		okID := h.enqueue(t.Context(), t, okArgs{Marker: uuid.New()}, nil)
		failID := h.enqueue(t.Context(), t, failArgs{Marker: uuid.New()}, &river.InsertOpts{MaxAttempts: 2})
		h.waitState(t, okID, "completed")
		h.waitState(t, failID, "discarded")
		// The metric is recorded after the job's state is saved.
		h.spanFor(t, "jobs.test_fail", failID)

		after := h.collect(t)
		delta := func(name, kind string, extra ...func(metricdata.DataPoint[int64]) bool) int64 {
			return sumFor(after[name], kind, extra...) - sumFor(before[name], kind, extra...)
		}
		discarded := func(want bool) func(metricdata.DataPoint[int64]) bool {
			return func(dp metricdata.DataPoint[int64]) bool {
				v, ok := dp.Attributes.Value("job.discarded")
				return ok && v.AsBool() == want
			}
		}

		require.Equal(t, int64(1), delta("cuadra.jobs.enqueued", "test_ok"))
		require.Equal(t, int64(1), delta("cuadra.jobs.enqueued", "test_fail"))
		require.Equal(t, int64(1), delta("cuadra.jobs.completed", "test_ok"))
		require.Equal(t, int64(1), delta("cuadra.jobs.failed", "test_fail", discarded(false)))
		require.Equal(t, int64(1), delta("cuadra.jobs.failed", "test_fail", discarded(true)))

		for _, name := range []string{"cuadra.jobs.queue_latency", "cuadra.jobs.duration"} {
			m, ok := after[name]
			require.True(t, ok, "falta la métrica %s", name)
			require.Equal(t, "s", m.Unit)
			hist, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok, "%s debe ser un histograma", name)
			require.NotEmpty(t, hist.DataPoints)
		}
	})
}
