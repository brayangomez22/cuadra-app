// Package app holds the feedback use cases: a user sends a comment from the
// app, and a background job tells the team about it.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback"

// Actor is who runs a use case: ids from the access token.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Notice is what the team is told about a comment.
type Notice struct {
	FeedbackID uuid.UUID
	TenantID   uuid.UUID
	Screen     string
	AppVersion string
	Message    string
	CreatedAt  time.Time
}

// Notifier tells the team about a comment (a webhook of its chat, or the
// log when none is configured).
type Notifier interface {
	Notify(ctx context.Context, n Notice) error
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Feedback       domain.Repository
	Notifier       Notifier
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the feedback use cases.
type Service struct {
	Deps
	spans         usecase.Spans
	received      metric.Int64Counter
	notifications metric.Int64Counter
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Feedback, d.Notifier) {
		return nil, errors.New("feedback: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "feedback")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.received, "cuadra.feedback.received",
			"Comments users sent from the app.", "{comment}"),
		usecase.Counter(&s.notifications, "cuadra.feedback.notifications",
			"Notices of a comment to the team, by result (sent, failed).", "{notice}"),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the tenant's id and attrs.
func (s *Service) startSpan(ctx context.Context, name string, tenantID uuid.UUID, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.spans.Start(ctx, name, append(attrs, attribute.String("tenant.id", tenantID.String()))...)
}
