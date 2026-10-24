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

// GetSale returns a sale with its lines and payments.
func (s *Service) GetSale(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Sale, err error) {
	ctx, span := s.startSpan(ctx, "GetSale", a, saleAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Sales.Get(ctx, a.TenantID, id)
}

// ListSales returns one page of the sales confirmed on a day in Colombia,
// voided ones included, newest first.
func (s *Service) ListSales(ctx context.Context, a Actor, in ListSalesInput) (_ domain.SalePage, err error) {
	ctx, span := s.startSpan(ctx, "ListSales", a)
	defer func() { usecase.End(span, err) }()

	day := in.Date
	if day.IsZero() {
		day = s.Now().In(bogota)
	}
	y, m, d := day.Date()
	from := time.Date(y, m, d, 0, 0, 0, 0, bogota)
	return s.Sales.List(ctx, a.TenantID, domain.SaleFilter{
		From: from.UTC(), To: from.AddDate(0, 0, 1).UTC(),
		LocationID: in.LocationID, Limit: in.Limit, Offset: in.Offset,
	})
}
