package demo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/demo"

// resetHour is when the demo tenants are reset each night, in Bogotá.
const resetHour = 3

// resetTimeout is how long a reset of every demo may take: a full one
// writes some twenty thousand sales.
const resetTimeout = time.Hour

// Resetter regenerates the demo tenants: it deletes the current demo of a
// template and writes its scenario again, with the history ending
// yesterday. The same Options always give the same tenant.
type Resetter struct {
	Generator Generator
	Templates []string
	Password  string
	// Options scale the scenarios; their End is ignored (yesterday).
	Options Options
	// Now is the real clock (time.Now by default): it picks yesterday.
	Now func() time.Time

	spans    usecase.Spans
	resets   metric.Int64Counter
	duration metric.Float64Histogram
}

// NewResetter checks r and creates its telemetry.
func NewResetter(r Resetter) (*Resetter, error) {
	if r.Generator.DB == nil || r.Generator.Jobs == nil || r.Password == "" || len(r.Templates) == 0 {
		return nil, errors.New("demo: the resetter needs the database, the job client, a password and templates")
	}
	if r.Now == nil {
		r.Now = time.Now
	}
	r.Generator.Log = usecase.Logger(r.Generator.Log)
	r.spans = usecase.NewSpans(r.Generator.TracerProvider, instrumentationName, "demo")
	err := usecase.Create(usecase.Meter(r.Generator.MeterProvider, instrumentationName),
		usecase.Counter(&r.resets, "cuadra.demo.resets",
			"Demo tenants reset, by business template and result (completed, failed).", "{reset}"),
		usecase.Histogram(&r.duration, "cuadra.demo.reset_duration",
			"Time to delete a demo tenant and generate it again.", "s", 10, 30, 60, 120, 300, 600, 1200, 1800),
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ResetAll resets every template's demo.
func (r *Resetter) ResetAll(ctx context.Context) error {
	var errs []error
	for _, t := range r.Templates {
		if _, err := r.Reset(ctx, t); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Reset deletes the demo tenants of template and generates it again; it
// returns the new tenant's id.
func (r *Resetter) Reset(ctx context.Context, template string) (_ uuid.UUID, err error) {
	ctx, span := r.spans.Start(ctx, "Reset", attribute.String("business_template", template))
	start := time.Now()
	defer func() {
		usecase.End(span, err)
		result := "completed"
		if err != nil {
			result = "failed"
		}
		attrs := metric.WithAttributes(attribute.String("business_template", template), attribute.String("result", result))
		r.resets.Add(ctx, 1, attrs)
		r.duration.Record(ctx, time.Since(start).Seconds(), attrs)
	}()

	s, err := r.scenario(template)
	if err != nil {
		return uuid.Nil, err
	}
	if err := r.purge(ctx, template); err != nil {
		return uuid.Nil, err
	}
	tenant, err := r.generate(ctx, s)
	if err != nil {
		return uuid.Nil, err
	}
	span.SetAttributes(attribute.String("tenant.id", tenant.String()))
	r.Generator.Log.InfoContext(ctx, "demo tenant reset",
		slog.String("tenant_id", tenant.String()), slog.String("business_template", template))
	return tenant, nil
}

// scenario is template's scenario with the history ending yesterday.
func (r *Resetter) scenario(template string) (Scenario, error) {
	p, err := LoadProfile(template)
	if err != nil {
		return Scenario{}, err
	}
	o := r.Options
	o.End = r.Now().In(bogota).AddDate(0, 0, -1)
	return NewScenario(p, o)
}

// purge deletes every demo tenant of template (normally one).
func (r *Resetter) purge(ctx context.Context, template string) error {
	_, d, err := r.Generator.newAPI(&Clock{})
	if err != nil {
		return err
	}
	tenants, err := d.Tenants(ctx, template)
	if err != nil {
		return err
	}
	for _, id := range tenants {
		if err := d.Purge(ctx, id); err != nil {
			return fmt.Errorf("demo: purge %s: %w", id, err)
		}
	}
	return nil
}

// generate writes s under the demo.Generate span.
func (r *Resetter) generate(ctx context.Context, s Scenario) (_ uuid.UUID, err error) {
	ctx, span := r.spans.Start(ctx, "Generate", attribute.String("business_template", s.Profile.Template))
	defer func() { usecase.End(span, err) }()
	return r.Generator.Generate(ctx, s, r.Password)
}

// Fingerprint reads template's demo tenant as Reset left it.
func (r *Resetter) Fingerprint(ctx context.Context, tenant uuid.UUID, template string) (Fingerprint, error) {
	s, err := r.scenario(template)
	if err != nil {
		return Fingerprint{}, err
	}
	return r.Generator.Fingerprint(ctx, tenant, s, r.Password)
}

// ResetArgs are the arguments of the nightly job that resets every demo.
type ResetArgs struct{}

// Kind names the job in River.
func (ResetArgs) Kind() string { return "demo.reset" }

// ResetWorker works the ResetArgs jobs.
type ResetWorker struct {
	river.WorkerDefaults[ResetArgs]
	Resetter *Resetter
}

// Work resets the demos; River retries it if one failed (resetting again
// deletes what a failed attempt left).
func (w *ResetWorker) Work(ctx context.Context, _ *river.Job[ResetArgs]) error {
	return w.Resetter.ResetAll(ctx)
}

// Timeout is the most one attempt may take.
func (w *ResetWorker) Timeout(*river.Job[ResetArgs]) time.Duration { return resetTimeout }

// Schedule adds the reset worker to workers and its nightly schedule
// (resetHour in Bogotá) to the job client.
func (r *Resetter) Schedule(workers *river.Workers, client *jobs.Client) error {
	if err := river.AddWorkerSafely(workers, &ResetWorker{Resetter: r}); err != nil {
		return err
	}
	return client.AddPeriodic(river.NewPeriodicJob(jobs.Daily(resetHour, 0, jobs.Bogota),
		func() (river.JobArgs, *river.InsertOpts) { return ResetArgs{}, nil },
		&river.PeriodicJobOpts{ID: ResetArgs{}.Kind()}))
}
