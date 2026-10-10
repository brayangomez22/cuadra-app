package app

import (
	"context"

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

// ListStock returns one page of the levels at a location;
// ErrLocationNotFound if it does not exist in the actor's tenant.
func (s *Service) ListStock(ctx context.Context, a Actor, f domain.StockFilter) (_ domain.StockLevelPage, err error) {
	ctx, span := s.startSpan(ctx, "ListStock", a, locationAttr(f.LocationID))
	defer func() { endSpan(span, err) }()

	if _, err := s.Locations.GetByID(ctx, a.TenantID, f.LocationID); err != nil {
		return domain.StockLevelPage{}, err
	}
	f.Limit, f.Offset = pageBounds(f.Limit, f.Offset)
	return s.Stock.ListLevels(ctx, a.TenantID, f)
}

// GetKardex returns one page of a product's movements, newest first. A
// product without movements, or of another tenant, has an empty kardex.
func (s *Service) GetKardex(ctx context.Context, a Actor, f domain.KardexFilter) (_ domain.MovementPage, err error) {
	ctx, span := s.startSpan(ctx, "GetKardex", a, productAttr(f.ProductID))
	defer func() { endSpan(span, err) }()

	f.Limit, f.Offset = pageBounds(f.Limit, f.Offset)
	return s.Stock.Kardex(ctx, a.TenantID, f)
}
