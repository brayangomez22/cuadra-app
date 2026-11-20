// Package app holds the getting-started use cases: a user reads the
// checklist of their tenant, built from its real data, and hides it.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding"

// Actor is who runs a use case: ids and role from the access token.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     string
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	State      domain.StateReader
	Dismissals domain.DismissalRepository
	Templates  Templates
	Modules    Modules
	Invoicing  Invoicing
	// Can is whether a role has a permission (the identity matrix).
	Can            func(role, permission string) bool
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the getting-started use cases.
type Service struct {
	Deps
	spans      usecase.Spans
	dismissals metric.Int64Counter
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.State, d.Dismissals, d.Templates, d.Modules, d.Invoicing) || d.Can == nil {
		return nil, errors.New("onboarding: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "onboarding")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.dismissals, "cuadra.onboarding.dismissals",
			"Getting-started checklists hidden by their user.", "{dismissal}"),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's tenant and user.
func (s *Service) startSpan(ctx context.Context, name string, a Actor) (context.Context, trace.Span) {
	return s.spans.Start(ctx, name,
		attribute.String("tenant.id", a.TenantID.String()), attribute.String("user.id", a.UserID.String()))
}
