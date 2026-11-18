package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// bogota is the time zone of the business day. Colombia has been at UTC−5
// without daylight saving time since 1993, so a fixed zone does not depend
// on the tz database being installed in the container.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// ListSalesInput selects the sales of a day. Date's year, month and day are
// the day in Colombia; the zero Date is today. LocationID is optional.
type ListSalesInput struct {
	Date       time.Time
	LocationID *uuid.UUID
	Limit      int
	Offset     int
}

// GetSale returns a sale with its lines and payments. A user who does not
// see every sale gets only their own: another's is domain.ErrSaleNotFound.
func (s *Service) GetSale(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Sale, err error) {
	ctx, span := s.startSpan(ctx, "GetSale", a, saleAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.visibleSale(ctx, a, id)
}

func (s *Service) visibleSale(ctx context.Context, a Actor, id uuid.UUID) (*domain.Sale, error) {
	sale, err := s.Sales.Get(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if !a.SeesAllSales && sale.UserID() != a.UserID {
		return nil, domain.ErrSaleNotFound
	}
	return sale, nil
}

// SaleDetail is a sale with the names of who sold it and who voided it
// (empty for a user that no longer exists, or a sale not voided).
type SaleDetail struct {
	Sale         *domain.Sale
	CashierName  string
	VoidedByName string
}

// GetSaleDetail returns a visible sale with the names of its users, for
// its detail and its reprinted receipt.
func (s *Service) GetSaleDetail(ctx context.Context, a Actor, id uuid.UUID) (_ SaleDetail, err error) {
	ctx, span := s.startSpan(ctx, "GetSaleDetail", a, saleAttr(id))
	defer func() { usecase.End(span, err) }()

	sale, err := s.visibleSale(ctx, a, id)
	if err != nil {
		return SaleDetail{}, err
	}
	ids := []uuid.UUID{sale.UserID()}
	if v := sale.VoidInfo(); v != nil {
		ids = append(ids, v.UserID)
	}
	names, err := s.Users.UserNames(ctx, a.TenantID, ids)
	if err != nil {
		return SaleDetail{}, err
	}
	d := SaleDetail{Sale: sale, CashierName: names[sale.UserID()]}
	if v := sale.VoidInfo(); v != nil {
		d.VoidedByName = names[v.UserID]
	}
	return d, nil
}

// ListSales returns one page of the sales confirmed on a day in Colombia,
// voided ones included, newest first: all of them for a user who sees every
// sale, and only their own for anyone else (a cashier).
func (s *Service) ListSales(ctx context.Context, a Actor, in ListSalesInput) (_ domain.SalePage, err error) {
	ctx, span := s.startSpan(ctx, "ListSales", a)
	defer func() { usecase.End(span, err) }()

	day := in.Date
	if day.IsZero() {
		day = s.Now().In(bogota)
	}
	y, m, d := day.Date()
	from := time.Date(y, m, d, 0, 0, 0, 0, bogota)
	filter := domain.SaleFilter{
		From: from.UTC(), To: from.AddDate(0, 0, 1).UTC(),
		LocationID: in.LocationID, Limit: in.Limit, Offset: in.Offset,
	}
	if !a.SeesAllSales {
		filter.CashierID = &a.UserID
	}
	return s.Sales.List(ctx, a.TenantID, filter)
}
