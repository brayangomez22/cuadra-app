package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
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

// StockQuery selects the levels at a location, optionally of one variant.
type StockQuery struct {
	LocationID uuid.UUID
	VariantID  *uuid.UUID
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

// ListStock returns one page of the levels at a location;
// ErrLocationNotFound if it does not exist in the actor's tenant.
func (s *Service) ListStock(ctx context.Context, a Actor, q StockQuery) (_ domain.StockLevelPage, err error) {
	ctx, span := s.startSpan(ctx, "ListStock", a, locationAttr(q.LocationID))
	defer func() { endSpan(span, err) }()

	if _, err := s.Locations.GetByID(ctx, a.TenantID, q.LocationID); err != nil {
		return domain.StockLevelPage{}, err
	}
	f := domain.StockFilter{LocationID: q.LocationID, VariantID: q.VariantID}
	f.Limit, f.Offset = pageBounds(q.Limit, q.Offset)
	return s.Stock.ListLevels(ctx, a.TenantID, f)
}

// GetKardex returns one page of a variant's movements, newest first. A
// variant without movements, or of another tenant, has an empty kardex.
func (s *Service) GetKardex(ctx context.Context, a Actor, q KardexQuery) (_ domain.MovementPage, err error) {
	ctx, span := s.startSpan(ctx, "GetKardex", a, variantAttr(q.VariantID))
	defer func() { endSpan(span, err) }()

	f := domain.KardexFilter{VariantID: q.VariantID, LocationID: q.LocationID}
	f.Limit, f.Offset = pageBounds(q.Limit, q.Offset)
	return s.Stock.Kardex(ctx, a.TenantID, f)
}

// ProductsOf returns the product of each variant of the actor's tenant among
// variantIDs, for the product_id of the levels and movements.
func (s *Service) ProductsOf(ctx context.Context, a Actor, variantIDs []uuid.UUID) (_ map[uuid.UUID]uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "ProductsOf", a)
	defer func() { endSpan(span, err) }()

	if len(variantIDs) == 0 {
		return map[uuid.UUID]uuid.UUID{}, nil
	}
	return s.Variants.ProductsOf(ctx, a.TenantID, variantIDs)
}
