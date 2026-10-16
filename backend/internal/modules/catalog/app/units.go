package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// defaultUnits are the units every new tenant starts with: the fixed list
// the catalog had before units belonged to each tenant, the same that
// migrations/00014_units_of_measure.sql gives the existing tenants. T24g
// replaces them with the units of the tenant's business template.
var defaultUnits = []struct{ code, name string }{
	{"und", "Unidad"},
	{"m", "Metro"},
	{"kg", "Kilogramo"},
	{"l", "Litro"},
	{"caja", "Caja"},
	{"rollo", "Rollo"},
	{"bulto", "Bulto"},
	{"galon", "Galón"},
}

// CreateUnit stores a new unit of measure.
func (s *Service) CreateUnit(ctx context.Context, a Actor, code, name string) (_ *domain.UnitOfMeasure, err error) {
	ctx, span := s.startSpan(ctx, "CreateUnit", a)
	defer func() { endSpan(span, err) }()

	u, err := domain.NewUnit(a.TenantID, code, name, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(unitAttr(u.ID()))
	if err := s.Units.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUnit changes a unit's code and name. The products that use it keep
// using it: they reference its id.
func (s *Service) UpdateUnit(ctx context.Context, a Actor, id uuid.UUID, code, name string) (_ *domain.UnitOfMeasure, err error) {
	ctx, span := s.startSpan(ctx, "UpdateUnit", a, unitAttr(id))
	defer func() { endSpan(span, err) }()

	u, err := s.Units.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.Update(code, name); err != nil {
		return nil, err
	}
	if err := s.Units.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// ListUnits returns every unit of the tenant, by name.
func (s *Service) ListUnits(ctx context.Context, a Actor) (_ []*domain.UnitOfMeasure, err error) {
	ctx, span := s.startSpan(ctx, "ListUnits", a)
	defer func() { endSpan(span, err) }()

	return s.Units.List(ctx, a.TenantID)
}

// DeleteUnit removes a unit no product uses.
func (s *Service) DeleteUnit(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteUnit", a, unitAttr(id))
	defer func() { endSpan(span, err) }()

	return s.Units.Delete(ctx, a.TenantID, id)
}

// SeedDefaultUnits gives a new tenant its initial units of measure, all or
// none. It runs at sign-up, before the tenant has a user acting on it.
func (s *Service) SeedDefaultUnits(ctx context.Context, tenantID uuid.UUID) (err error) {
	ctx, span := s.tracer.Start(ctx, "catalog.SeedDefaultUnits",
		trace.WithAttributes(attribute.String("tenant.id", tenantID.String())))
	defer func() { endSpan(span, err) }()

	units := make([]*domain.UnitOfMeasure, 0, len(defaultUnits))
	for _, d := range defaultUnits {
		u, err := domain.NewUnit(tenantID, d.code, d.name, s.Now())
		if err != nil {
			return err
		}
		units = append(units, u)
	}
	return s.Units.CreateMany(ctx, tenantID, units)
}
