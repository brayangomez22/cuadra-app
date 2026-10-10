package app

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// CreateCategory stores a new category under parentID, or at the root.
func (s *Service) CreateCategory(ctx context.Context, a Actor, name string, parentID *uuid.UUID) (_ *domain.Category, err error) {
	ctx, span := s.startSpan(ctx, "CreateCategory", a)
	defer func() { endSpan(span, err) }()

	c, err := domain.NewCategory(a.TenantID, name, parentID, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(categoryAttr(c.ID()))
	if err := s.Categories.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateCategory renames the category and moves it under parentID (the root
// when nil). It refuses to move a category under itself or one of its
// descendants. Two concurrent moves could still build a cycle; the risk is
// accepted (docs/decisiones.md) and Ancestors stops walking a cycle.
func (s *Service) UpdateCategory(ctx context.Context, a Actor, id uuid.UUID, name string, parentID *uuid.UUID) (_ *domain.Category, err error) {
	ctx, span := s.startSpan(ctx, "UpdateCategory", a, categoryAttr(id))
	defer func() { endSpan(span, err) }()

	c, err := s.Categories.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := c.Rename(name); err != nil {
		return nil, err
	}
	if err := c.MoveTo(parentID); err != nil {
		return nil, err
	}
	if parentID != nil {
		ancestors, err := s.Categories.Ancestors(ctx, a.TenantID, *parentID)
		if err != nil {
			return nil, err
		}
		if len(ancestors) == 0 {
			return nil, domain.ErrCategoryNotFound
		}
		if slices.Contains(ancestors, id) {
			return nil, domain.ErrCategoryCycle
		}
	}
	if err := s.Categories.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCategories returns every category of the tenant, by name.
func (s *Service) ListCategories(ctx context.Context, a Actor) (_ []*domain.Category, err error) {
	ctx, span := s.startSpan(ctx, "ListCategories", a)
	defer func() { endSpan(span, err) }()

	return s.Categories.List(ctx, a.TenantID)
}

// DeleteCategory removes a category without products or subcategories.
func (s *Service) DeleteCategory(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteCategory", a, categoryAttr(id))
	defer func() { endSpan(span, err) }()

	return s.Categories.Delete(ctx, a.TenantID, id)
}
