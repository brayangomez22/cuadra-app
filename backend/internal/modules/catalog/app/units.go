package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// UnitInput is a unit's code, name and DIAN code (empty: none).
type UnitInput struct{ Code, Name, DIANCode string }

// CreateUnit stores a new unit of measure.
func (s *Service) CreateUnit(ctx context.Context, a Actor, in UnitInput) (_ *domain.UnitOfMeasure, err error) {
	ctx, span := s.startSpan(ctx, "CreateUnit", a)
	defer func() { usecase.End(span, err) }()

	u, err := domain.NewUnit(a.TenantID, in.Code, in.Name, s.Now())
	if err != nil {
		return nil, err
	}
	if err := u.SetDIANCode(in.DIANCode); err != nil {
		return nil, err
	}
	span.SetAttributes(unitAttr(u.ID()))
	if err := s.Units.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUnit changes a unit's code, name and DIAN code. The products that
// use it keep using it: they reference its id. The lines already sold keep
// the codes they froze.
func (s *Service) UpdateUnit(ctx context.Context, a Actor, id uuid.UUID, in UnitInput) (_ *domain.UnitOfMeasure, err error) {
	ctx, span := s.startSpan(ctx, "UpdateUnit", a, unitAttr(id))
	defer func() { usecase.End(span, err) }()

	u, err := s.Units.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.Update(in.Code, in.Name); err != nil {
		return nil, err
	}
	if err := u.SetDIANCode(in.DIANCode); err != nil {
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
	defer func() { usecase.End(span, err) }()

	return s.Units.List(ctx, a.TenantID)
}

// DeleteUnit removes a unit no product uses.
func (s *Service) DeleteUnit(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteUnit", a, unitAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Units.Delete(ctx, a.TenantID, id)
}
