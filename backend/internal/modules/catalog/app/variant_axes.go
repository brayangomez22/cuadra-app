package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CreateVariantAxis stores a new variant axis with the values it allows.
func (s *Service) CreateVariantAxis(ctx context.Context, a Actor, name string, values []string) (_ *domain.VariantAxis, err error) {
	ctx, span := s.startSpan(ctx, "CreateVariantAxis", a)
	defer func() { usecase.End(span, err) }()

	axis, err := domain.NewVariantAxis(a.TenantID, name, values, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(axisAttr(axis.ID()))
	if err := s.VariantAxes.Create(ctx, axis); err != nil {
		return nil, err
	}
	return axis, nil
}

// UpdateVariantAxis replaces the values an axis allows. A value a variant
// uses cannot be removed. The name never changes: variants keep their
// values under it.
func (s *Service) UpdateVariantAxis(ctx context.Context, a Actor, id uuid.UUID, values []string) (_ *domain.VariantAxis, err error) {
	ctx, span := s.startSpan(ctx, "UpdateVariantAxis", a, axisAttr(id))
	defer func() { usecase.End(span, err) }()

	axis, err := s.VariantAxes.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	removed, err := axis.Update(values)
	if err != nil {
		return nil, err
	}
	if err := s.VariantAxes.Update(ctx, axis, removed); err != nil {
		return nil, err
	}
	return axis, nil
}

// ListVariantAxes returns every variant axis of the tenant, by name.
func (s *Service) ListVariantAxes(ctx context.Context, a Actor) (_ []*domain.VariantAxis, err error) {
	ctx, span := s.startSpan(ctx, "ListVariantAxes", a)
	defer func() { usecase.End(span, err) }()

	return s.VariantAxes.List(ctx, a.TenantID)
}

// DeleteVariantAxis removes an axis no variant uses.
func (s *Service) DeleteVariantAxis(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteVariantAxis", a, axisAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.VariantAxes.Delete(ctx, a.TenantID, id)
}

// resolveAxisValues checks a variant's axis values against the tenant's axes.
func (s *Service) resolveAxisValues(ctx context.Context, a Actor, in map[string]string) (map[string]string, error) {
	axes, err := s.VariantAxes.List(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	return domain.ResolveAxisValues(axes, in)
}
