package modules

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"

// Change turns one module on or off for a tenant.
type Change struct {
	Module string
	Active bool
	// By is the user who made the change, and At when.
	By uuid.UUID
	At time.Time
}

// Store keeps the active modules of each tenant.
type Store interface {
	// Active lists the IDs a tenant has active, in no particular order. It
	// may include IDs that are no longer registered.
	Active(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	// Update runs decide with the tenant's active modules and applies the
	// change it returns (none when it returns nil). One tenant's updates run
	// one at a time, so decide sees the state its change applies to. When
	// decide fails, nothing is written and its error is returned as is.
	Update(ctx context.Context, tenantID uuid.UUID, decide func(active []string) (*Change, error)) error
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Registry       *Registry
	Store          Store
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Status is a module and whether a tenant has it active.
type Status struct {
	Definition
	Active bool
}

// Service turns the registered modules on and off for each tenant.
type Service struct {
	Deps
	tracer  trace.Tracer
	changes metric.Int64Counter
}

// NewService builds a Service. It fails if the registry or the store is
// missing, or the metric cannot be created.
func NewService(d Deps) (*Service, error) {
	if d.Registry == nil || d.Store == nil {
		return nil, errors.New("modules: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.TracerProvider == nil {
		d.TracerProvider = tracenoop.NewTracerProvider()
	}
	if d.MeterProvider == nil {
		d.MeterProvider = metricnoop.NewMeterProvider()
	}
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}
	s := &Service{Deps: d, tracer: d.TracerProvider.Tracer(instrumentationName)}
	var err error
	if s.changes, err = d.MeterProvider.Meter(instrumentationName).Int64Counter("cuadra.modules.changes",
		metric.WithDescription("Modules turned on or off for a tenant, by module and action (enabled, disabled)."),
		metric.WithUnit("{change}")); err != nil {
		return nil, err
	}
	return s, nil
}

// Known reports whether id is a registered module.
func (s *Service) Known(id string) bool { return s.Registry.Known(id) }

// Active lists the registered modules the tenant has active, in registration
// order. It runs on every GET /me and every operation of a module, so it
// opens no span of its own: the query's span is enough.
func (s *Service) Active(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	stored, err := s.Store.Active(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.registered(stored), nil
}

// IsActive reports whether the tenant has the module id active. An
// unregistered module is never active.
func (s *Service) IsActive(ctx context.Context, tenantID uuid.UUID, id string) (bool, error) {
	if !s.Registry.Known(id) {
		return false, nil
	}
	active, err := s.Store.Active(ctx, tenantID)
	if err != nil {
		return false, err
	}
	return slices.Contains(active, id), nil
}

// List returns every registered module with whether the tenant has it active.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) (_ []Status, err error) {
	ctx, span := s.tracer.Start(ctx, "modules.ListModules")
	defer func() { endSpan(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()))

	active, err := s.Store.Active(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defs := s.Registry.Definitions()
	out := make([]Status, len(defs))
	for i, d := range defs {
		out[i] = Status{Definition: d, Active: slices.Contains(active, d.ID)}
	}
	return out, nil
}

// SetActive turns the module id on or off for the tenant, on behalf of
// userID. It is idempotent. A module is turned on only when its
// dependencies are active, and off only when no active module depends on
// it; otherwise it fails with a *DependencyError. Turning a module off
// keeps its data.
func (s *Service) SetActive(ctx context.Context, tenantID, userID uuid.UUID, id string, active bool) (_ Status, err error) {
	name := "modules.DisableModule"
	if active {
		name = "modules.EnableModule"
	}
	ctx, span := s.tracer.Start(ctx, name)
	defer func() { endSpan(span, err) }()
	span.SetAttributes(
		attribute.String("tenant.id", tenantID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("module.id", id),
	)

	def, ok := s.Registry.get(id)
	if !ok {
		return Status{}, ErrUnknownModule
	}
	changed := false
	err = s.Store.Update(ctx, tenantID, func(current []string) (*Change, error) {
		current = s.registered(current)
		if slices.Contains(current, id) == active {
			return nil, nil
		}
		if err := s.checkDependencies(def, current, active); err != nil {
			return nil, err
		}
		changed = true
		return &Change{Module: id, Active: active, By: userID, At: s.Now().UTC()}, nil
	})
	if err != nil {
		return Status{}, err
	}

	if changed {
		action := "disabled"
		if active {
			action = "enabled"
		}
		s.changes.Add(ctx, 1, metric.WithAttributes(attribute.String("module.id", id), attribute.String("action", action)))
		s.Logger.InfoContext(ctx, "module "+action, slog.String("module_id", id))
	}
	return Status{Definition: def.clone(), Active: active}, nil
}

// checkDependencies checks that def can be turned on (its dependencies are
// active) or off (no active module depends on it), given the active ones.
func (s *Service) checkDependencies(def Definition, active []string, enable bool) error {
	if enable {
		for _, dep := range def.DependsOn {
			if !slices.Contains(active, dep) {
				return s.dependencyError(ErrDependencyInactive, dep)
			}
		}
		return nil
	}
	for _, id := range active {
		if d, _ := s.Registry.get(id); slices.Contains(d.DependsOn, def.ID) {
			return s.dependencyError(ErrRequiredBy, id)
		}
	}
	return nil
}

func (s *Service) dependencyError(err error, id string) *DependencyError {
	d, _ := s.Registry.get(id)
	return &DependencyError{Err: err, Module: id, Name: d.Name}
}

// registered keeps the registered modules of ids, in registration order.
func (s *Service) registered(ids []string) []string {
	out := []string{}
	for _, d := range s.Registry.defs {
		if slices.Contains(ids, d.ID) {
			out = append(out, d.ID)
		}
	}
	return out
}

func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
