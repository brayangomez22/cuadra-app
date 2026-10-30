package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Page size bounds of the queries.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// pageBounds applies the default page size, the cap of 100 and a
// non-negative offset.
func pageBounds(limit, offset int) (int, int) {
	switch {
	case limit <= 0:
		limit = defaultPageSize
	case limit > maxPageSize:
		limit = maxPageSize
	}
	return limit, max(offset, 0)
}

// StockQuery selects the levels at a location, optionally of one variant or
// of some (VariantIDs, when not nil: the ones a catalog search found).
type StockQuery struct {
	LocationID uuid.UUID
	VariantID  *uuid.UUID
	VariantIDs []uuid.UUID
	Limit      int
	Offset     int
}

// KardexQuery selects a variant's movements, optionally at one location.
type KardexQuery struct {
	VariantID  uuid.UUID
	LocationID *uuid.UUID
	Limit      int
	Offset     int
}

// ListStock returns one page of the levels at a location, each with its
// reorder rule;
// ErrLocationNotFound if it does not exist in the actor's tenant.
func (s *Service) ListStock(ctx context.Context, a Actor, q StockQuery) (_ LevelPage, err error) {
	ctx, span := s.startSpan(ctx, "ListStock", a, locationAttr(q.LocationID))
	defer func() { usecase.End(span, err) }()

	if _, err := s.Locations.GetByID(ctx, a.TenantID, q.LocationID); err != nil {
		return LevelPage{}, err
	}
	f := domain.StockFilter{LocationID: q.LocationID, VariantID: q.VariantID, VariantIDs: q.VariantIDs}
	f.Limit, f.Offset = pageBounds(q.Limit, q.Offset)
	page, err := s.Stock.ListLevels(ctx, a.TenantID, f)
	if err != nil {
		return LevelPage{}, err
	}
	items, err := s.levelsWithProducts(ctx, a, page.Items)
	if err != nil {
		return LevelPage{}, err
	}
	if err := s.withReorder(ctx, a, q.LocationID, items); err != nil {
		return LevelPage{}, err
	}
	return LevelPage{Items: items, Total: page.Total}, nil
}

// GetKardex returns one page of a variant's movements, newest first. A
// variant without movements, or of another tenant, has an empty kardex.
func (s *Service) GetKardex(ctx context.Context, a Actor, q KardexQuery) (_ MovementPage, err error) {
	ctx, span := s.startSpan(ctx, "GetKardex", a, variantAttr(q.VariantID))
	defer func() { usecase.End(span, err) }()

	f := domain.KardexFilter{VariantID: q.VariantID, LocationID: q.LocationID}
	f.Limit, f.Offset = pageBounds(q.Limit, q.Offset)
	page, err := s.Stock.Kardex(ctx, a.TenantID, f)
	if err != nil {
		return MovementPage{}, err
	}
	items, err := s.withProducts(ctx, a, page.Items...)
	if err != nil {
		return MovementPage{}, err
	}
	return MovementPage{Items: items, Total: page.Total}, nil
}

// ProductsOf returns the product of each variant of the actor's tenant among
// variantIDs, for the product of the levels and movements the use cases
// return.
func (s *Service) ProductsOf(ctx context.Context, a Actor, variantIDs []uuid.UUID) (_ map[uuid.UUID]uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "ProductsOf", a)
	defer func() { usecase.End(span, err) }()

	if len(variantIDs) == 0 {
		return map[uuid.UUID]uuid.UUID{}, nil
	}
	return s.Variants.ProductsOf(ctx, a.TenantID, variantIDs)
}
