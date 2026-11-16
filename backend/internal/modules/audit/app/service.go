// Package app holds the audit use cases: recording an event of an audited
// operation (the other modules call it through a port, inside the
// operation's transaction) and reading the log.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/audit"

// Actor is who runs a use case: ids and role from the access token.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     string
}

// Deps are the Service's dependencies.
type Deps struct {
	Events         domain.EventRepository
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the audit use cases.
type Service struct {
	Deps
	spans  usecase.Spans
	events metric.Int64Counter
}

// NewService builds a Service. It fails if the repository is missing or a
// metric instrument cannot be created. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Events) {
		return nil, errors.New("audit: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "audit")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.events, "cuadra.audit.events",
			"Audit events recorded, by action.", "{event}"),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.spans.StartFor(ctx, name, a.TenantID, a.UserID, attrs...)
}
