package flags

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"

// ErrNotOperator is the error for a user who is not an operator of the
// platform: only operators see and change the flags.
var ErrNotOperator = errors.New("flags: not a platform operator")

// ServiceDeps are the Service's dependencies. Operators are the users of
// the team who manage the flags (PLATFORM_OPERATORS). Now defaults to
// time.Now, the providers to no-ops and Logger to a discarding logger.
type ServiceDeps struct {
	Evaluator      *Evaluator
	Operators      []uuid.UUID
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Status is a flag with its global value and its exceptions.
type Status struct {
	Definition
	Global     bool
	Exceptions []Exception
}

// Service lets the platform's operators see and change the flags.
type Service struct {
	ServiceDeps
	spans   usecase.Spans
	changes metric.Int64Counter
}

// NewService builds a Service.
func NewService(d ServiceDeps) (*Service, error) {
	if usecase.AnyNil(d.Evaluator) {
		return nil, errors.New("flags: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{ServiceDeps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "flags")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.changes, "cuadra.flags.changes",
			"Feature flags changed by an operator, by flag, scope (global, tenant) and action (enabled, disabled, cleared).",
			"{change}"),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// IsOperator reports whether userID is an operator of the platform. It
// reads static configuration, so it opens no span.
func (s *Service) IsOperator(userID uuid.UUID) bool {
	return userID != uuid.Nil && slices.Contains(s.Operators, userID)
}

// Active lists the flags that are on for the tenant (Evaluator.Active). It
// runs on every GET /me, so it opens no span.
func (s *Service) Active(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return s.Evaluator.Active(ctx, tenantID)
}

// List returns every declared flag with its global value and the exceptions
// of each tenant, for an operator.
func (s *Service) List(ctx context.Context, userID uuid.UUID) (_ []Status, err error) {
	ctx, span := s.spans.Start(ctx, "ListFlags", attribute.String("user.id", userID.String()))
	defer func() { usecase.End(span, err) }()
	if !s.IsOperator(userID) {
		return nil, ErrNotOperator
	}
	globals, err := s.Evaluator.Store.Globals(ctx)
	if err != nil {
		return nil, err
	}
	exceptions, err := s.Evaluator.Store.AllExceptions(ctx)
	if err != nil {
		return nil, err
	}
	defs := s.Evaluator.Registry.Definitions()
	out := make([]Status, len(defs))
	for i, d := range defs {
		out[i] = Status{Definition: d, Global: globals[d.Key], Exceptions: []Exception{}}
		for _, ex := range exceptions {
			if ex.Key == d.Key {
				out[i].Exceptions = append(out[i].Exceptions, ex)
			}
		}
	}
	return out, nil
}

// SetGlobal sets the value of a flag for every tenant without an exception.
func (s *Service) SetGlobal(ctx context.Context, userID uuid.UUID, key string, enabled bool) (_ Status, err error) {
	ctx, span := s.spans.Start(ctx, "SetGlobal",
		attribute.String("user.id", userID.String()), attribute.String("flag", key))
	defer func() { usecase.End(span, err) }()
	def, err := s.check(userID, key)
	if err != nil {
		return Status{}, err
	}
	if err := s.Evaluator.Store.SetGlobal(ctx, s.change(userID, key, enabled)); err != nil {
		return Status{}, err
	}
	s.changed(ctx, key, "global", action(enabled))
	return Status{Definition: def, Global: enabled}, nil
}

// SetException sets the value of a flag for one tenant, over the global one.
func (s *Service) SetException(ctx context.Context, userID, tenantID uuid.UUID, key string, enabled bool) (err error) {
	ctx, span := s.spans.Start(ctx, "SetException", attribute.String("user.id", userID.String()),
		attribute.String("tenant.id", tenantID.String()), attribute.String("flag", key))
	defer func() { usecase.End(span, err) }()
	if _, err := s.check(userID, key); err != nil {
		return err
	}
	if err := s.Evaluator.Store.SetException(ctx, tenantID, s.change(userID, key, enabled)); err != nil {
		return err
	}
	s.changed(ctx, key, "tenant", action(enabled))
	return nil
}

// ClearException removes a tenant's exception: the tenant follows the
// global value again. It is idempotent.
func (s *Service) ClearException(ctx context.Context, userID, tenantID uuid.UUID, key string) (err error) {
	ctx, span := s.spans.Start(ctx, "ClearException", attribute.String("user.id", userID.String()),
		attribute.String("tenant.id", tenantID.String()), attribute.String("flag", key))
	defer func() { usecase.End(span, err) }()
	if _, err := s.check(userID, key); err != nil {
		return err
	}
	if err := s.Evaluator.Store.ClearException(ctx, tenantID, key); err != nil {
		return err
	}
	s.changed(ctx, key, "tenant", "cleared")
	return nil
}

// check verifies that userID is an operator and key a declared flag.
func (s *Service) check(userID uuid.UUID, key string) (Definition, error) {
	if !s.IsOperator(userID) {
		return Definition{}, ErrNotOperator
	}
	i := slices.IndexFunc(s.Evaluator.Registry.defs, func(d Definition) bool { return d.Key == key })
	if i < 0 {
		return Definition{}, ErrUnknownFlag
	}
	return s.Evaluator.Registry.defs[i], nil
}

func (s *Service) change(userID uuid.UUID, key string, enabled bool) Change {
	return Change{Key: key, Enabled: enabled, By: userID, At: s.Now().UTC()}
}

// changed drops this instance's cache, so the change shows at once here
// (other instances see it within the cache's TTL), and records it.
func (s *Service) changed(ctx context.Context, key, scope, act string) {
	s.Evaluator.Invalidate()
	s.changes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("flag", key), attribute.String("scope", scope), attribute.String("action", act)))
	s.Logger.InfoContext(ctx, "feature flag changed",
		slog.String("flag", key), slog.String("scope", scope), slog.String("action", act))
}

func action(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
