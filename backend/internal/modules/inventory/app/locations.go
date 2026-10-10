package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// CreateLocation stores a new, active location.
func (s *Service) CreateLocation(ctx context.Context, a Actor, name string) (_ *domain.Location, err error) {
	ctx, span := s.startSpan(ctx, "CreateLocation", a)
	defer func() { endSpan(span, err) }()

	l, err := domain.NewLocation(a.TenantID, name, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(locationAttr(l.ID()))
	if err := s.Locations.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// ListLocations returns every location of the tenant, by name.
func (s *Service) ListLocations(ctx context.Context, a Actor) (_ []*domain.Location, err error) {
	ctx, span := s.startSpan(ctx, "ListLocations", a)
	defer func() { endSpan(span, err) }()

	return s.Locations.List(ctx, a.TenantID)
}

// requireActiveLocation checks that the location exists in the actor's
// tenant and is in use.
func (s *Service) requireActiveLocation(ctx context.Context, a Actor, id uuid.UUID) error {
	l, err := s.Locations.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return err
	}
	return l.RequireActive()
}
