package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"

// traceContextKey is the job metadata key holding the W3C trace context of
// the code that enqueued the job.
const traceContextKey = "trace_context"

// secondsBuckets covers jobs and queue waits from milliseconds to an hour.
var secondsBuckets = []float64{0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300, 900, 3600}

// propagator is fixed, not the global one, so the metadata format does not
// change with the process's telemetry configuration.
var propagator = propagation.TraceContext{}

// telemetry is a River middleware: on insert it saves the caller's trace
// context in the job and counts the job; on work it opens the job's span,
// linked to that context, and records the outcome.
type telemetry struct {
	river.MiddlewareDefaults

	tracer       trace.Tracer
	enqueued     metric.Int64Counter
	completed    metric.Int64Counter
	failed       metric.Int64Counter
	queueLatency metric.Float64Histogram
	duration     metric.Float64Histogram
}

var (
	_ rivertype.JobInsertMiddleware = (*telemetry)(nil)
	_ rivertype.WorkerMiddleware    = (*telemetry)(nil)
)

func newTelemetry(tp trace.TracerProvider, mp metric.MeterProvider) (*telemetry, error) {
	t := &telemetry{tracer: tp.Tracer(instrumentationName)}
	meter := mp.Meter(instrumentationName)
	var err error
	if t.enqueued, err = meter.Int64Counter("cuadra.jobs.enqueued",
		metric.WithDescription("Jobs inserted. A job inserted in a transaction that rolls back is counted too."),
		metric.WithUnit("{job}")); err != nil {
		return nil, err
	}
	if t.completed, err = meter.Int64Counter("cuadra.jobs.completed",
		metric.WithDescription("Job attempts that succeeded."),
		metric.WithUnit("{job}")); err != nil {
		return nil, err
	}
	if t.failed, err = meter.Int64Counter("cuadra.jobs.failed",
		metric.WithDescription("Job attempts that failed; job.discarded tells whether the job will not run again."),
		metric.WithUnit("{job}")); err != nil {
		return nil, err
	}
	if t.queueLatency, err = meter.Float64Histogram("cuadra.jobs.queue_latency",
		metric.WithDescription("Time from when a job was due to when an attempt started."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(secondsBuckets...)); err != nil {
		return nil, err
	}
	if t.duration, err = meter.Float64Histogram("cuadra.jobs.duration",
		metric.WithDescription("Time a job attempt took to run."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(secondsBuckets...)); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *telemetry) InsertMany(ctx context.Context, params []*rivertype.JobInsertParams, doInner func(context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	if len(carrier) > 0 {
		for _, p := range params {
			md, err := withTraceContext(p.Metadata, carrier)
			if err != nil {
				return nil, err
			}
			p.Metadata = md
		}
	}

	results, err := doInner(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range params {
		t.enqueued.Add(ctx, 1, metric.WithAttributes(attribute.String("job.kind", p.Kind)))
	}
	return results, nil
}

func (t *telemetry) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) (err error) {
	kind := attribute.String("job.kind", job.Kind)
	start := time.Now()
	if job.AttemptedAt != nil {
		t.queueLatency.Record(ctx, job.AttemptedAt.Sub(job.ScheduledAt).Seconds(), metric.WithAttributes(kind))
	}

	// A new trace per attempt, linked to the one that enqueued the job: a retry
	// hours later does not stretch the request's trace.
	opts := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.Int64("job.id", job.ID),
			kind,
			attribute.Int("job.attempt", job.Attempt),
		),
	}
	if origin := originSpanContext(job.Metadata); origin.IsValid() {
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: origin}))
	}
	ctx, span := t.tracer.Start(ctx, "jobs."+job.Kind, opts...)
	defer func() {
		if r := recover(); r != nil {
			t.recordFailure(ctx, span, job, kind, fmt.Errorf("panic: %v", r))
			t.duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(kind))
			span.End()
			panic(r)
		}
		t.duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(kind))
		span.End()
	}()

	err = doInner(ctx)
	var snooze *river.JobSnoozeError
	switch {
	case err == nil:
		t.completed.Add(ctx, 1, metric.WithAttributes(kind))
	case errors.As(err, &snooze):
		// Postponed on purpose: not a failure and not an attempt.
	default:
		t.recordFailure(ctx, span, job, kind, err)
	}
	return err
}

func (t *telemetry) recordFailure(ctx context.Context, span trace.Span, job *rivertype.JobRow, kind attribute.KeyValue, err error) {
	var cancel *river.JobCancelError
	discarded := errors.As(err, &cancel) || job.Attempt >= job.MaxAttempts
	span.RecordError(err)
	span.SetStatus(codes.Error, "job failed")
	span.SetAttributes(attribute.Bool("job.discarded", discarded))
	t.failed.Add(ctx, 1, metric.WithAttributes(kind, attribute.Bool("job.discarded", discarded)))
}

// withTraceContext returns the job metadata with the trace context added.
func withTraceContext(metadata []byte, carrier propagation.MapCarrier) ([]byte, error) {
	md := map[string]any{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &md); err != nil {
			return nil, fmt.Errorf("jobs: decode metadata: %w", err)
		}
	}
	md[traceContextKey] = map[string]string(carrier)
	out, err := json.Marshal(md)
	if err != nil {
		return nil, fmt.Errorf("jobs: encode metadata: %w", err)
	}
	return out, nil
}

// originSpanContext reads the trace context saved when the job was enqueued.
// Missing or malformed metadata yields an invalid span context: the job still
// runs, only without the link.
func originSpanContext(metadata []byte) trace.SpanContext {
	var md struct {
		TraceContext map[string]string `json:"trace_context"`
	}
	if err := json.Unmarshal(metadata, &md); err != nil || md.TraceContext == nil {
		return trace.SpanContext{}
	}
	ctx := propagator.Extract(context.Background(), propagation.MapCarrier(md.TraceContext))
	return trace.SpanContextFromContext(ctx)
}
