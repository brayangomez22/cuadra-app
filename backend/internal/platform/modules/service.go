package modules

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
	// change it returns, if any. One tenant's updates run one at a time, so
	// decide sees the state its change applies to. When decide fails,
	// nothing is written and its error is returned as is.
	Update(ctx context.Context, tenantID uuid.UUID, decide Decide) error
}

// Decide chooses the change to apply to a tenant's active modules; ok is
// false when there is nothing to change.
type Decide func(active []string) (change Change, ok bool, err error)

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
	spans   usecase.Spans
	changes metric.Int64Counter
}

// NewService builds a Service. It fails if the registry or the store is
// missing, or the metric cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Registry, d.Store) {
		return nil, errors.New("modules: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "modules")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.changes, "cuadra.modules.changes",
			"Modules turned on or off for a tenant, by module and action (enabled, disabled).", "{change}"),
	)
	if err != nil {
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
	ctx, span := s.spans.Start(ctx, "ListModules")
	defer func() { usecase.End(span, err) }()
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
	name := "DisableModule"
	if active {
		name = "EnableModule"
	}
	ctx, span := s.spans.Start(ctx, name)
	defer func() { usecase.End(span, err) }()
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
	err = s.Store.Update(ctx, tenantID, func(current []string) (Change, bool, error) {
		current = s.registered(current)
		if slices.Contains(current, id) == active {
			return Change{}, false, nil
		}
		if err := s.checkDependencies(def, current, active); err != nil {
			return Change{}, false, err
		}
		changed = true
		return Change{Module: id, Active: active, By: userID, At: s.Now().UTC()}, true, nil
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
		s.Logger.InfoContext(ctx, "module changed", slog.String("module_id", id), slog.String("action", action))
	}
	return Status{Definition: def.clone(), Active: active}, nil
}

// SetUpTenant turns on the modules ids for a new tenant, as its business
// template declares them, with no user behind the change. It checks the
// whole set first (Registry.CheckSet), so it turns on all of them or none;
// with a ctx from db.InTenantTx, the store joins the sign-up's transaction.
// It is not counted in cuadra.modules.changes: no one changed the tenant's
// modules.
func (s *Service) SetUpTenant(ctx context.Context, tenantID uuid.UUID, ids []string) (err error) {
	ctx, span := s.spans.Start(ctx, "SetUpTenant")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()))

	if err := s.Registry.CheckSet(ids); err != nil {
		return err
	}
	at := s.Now().UTC()
	for _, id := range s.registered(ids) {
		err := s.Store.Update(ctx, tenantID, func(active []string) (Change, bool, error) {
			if slices.Contains(active, id) {
				return Change{}, false, nil
			}
			return Change{Module: id, Active: true, At: at}, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
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
