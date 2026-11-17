// Package feedback wires the feedback module: the comments users send from
// the app ("Danos tu opinión", T40) and the notice to the team, which a
// background job sends. internal/server builds it with New and mounts its
// routes.
package feedback

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/http"
	jobsadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/notify"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Jobs enqueues the notices; the module adds its worker to Workers.
	Jobs    *jobs.Client
	Workers *river.Workers
	// WebhookURL is the team's chat webhook (FEEDBACK_WEBHOOK_URL); empty
	// writes the notices in the log.
	WebhookURL     string
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Now            func() time.Time
}

// Module is the wired feedback module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL and River.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Jobs == nil || cfg.Workers == nil || cfg.Logger == nil {
		return nil, errors.New("feedback: missing module dependency")
	}
	var notifier app.Notifier = notify.Log{Logger: cfg.Logger}
	if cfg.WebhookURL != "" {
		notifier = notify.NewWebhook(cfg.WebhookURL)
	}
	svc, err := app.NewService(app.Deps{
		Feedback:       postgres.NewFeedbackRepository(cfg.DB, jobsadapter.Enqueuer(cfg.Jobs)),
		Notifier:       notifier,
		Now:            cfg.Now,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	if err := river.AddWorkerSafely(cfg.Workers, &jobsadapter.NotifyWorker{Notifier: svc}); err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "feedback") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
