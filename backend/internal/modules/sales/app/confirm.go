package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// LineRequest is a line of the cart: a product, how much and the discount
// (an amount of COP before IVA). The price and the IVA come from the
// catalog, never from the request.
type LineRequest struct {
	ProductID uuid.UUID
	Quantity  decimal.Decimal
	Discount  decimal.Decimal
}

// PaymentRequest is a payment of the cart; Method is a domain.PaymentMethod
// code.
type PaymentRequest struct {
	Method string
	Amount decimal.Decimal
}

// ConfirmSaleInput is a whole cart, checked out at once. CustomerID is
// optional.
type ConfirmSaleInput struct {
	LocationID uuid.UUID
	CustomerID *uuid.UUID
	Lines      []LineRequest
	Payments   []PaymentRequest
}

// ConfirmSale builds the sale from the cart with the catalog's prices,
// checks the discounts against the actor's limit and the payments against
// the total, and stores it together with its stock exits and its
// consecutive number. The sale is charged in the actor's open cash session,
// which must be at the sale's location; without one, the sale is rejected
// unless the tenant lets users sell without it. If any step fails, nothing
// is stored.
func (s *Service) ConfirmSale(ctx context.Context, a Actor, in ConfirmSaleInput) (_ *domain.Sale, err error) {
	ctx, span := s.startSpan(ctx, "ConfirmSale", a, attribute.String("location.id", in.LocationID.String()))
	defer func() { endSpan(span, err) }()

	if len(in.Lines) == 0 {
		return nil, domain.ErrSaleWithoutLines
	}
	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	session, err := s.saleCashSession(ctx, a, policy, in.LocationID)
	if err != nil {
		return nil, err
	}
	products, err := s.products(ctx, a.TenantID, in.Lines)
	if err != nil {
		return nil, err
	}
	limit := discountLimit(a, policy)
	payments, err := parsePayments(in.Payments)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	span.SetAttributes(saleAttr(id))
	params := domain.NewSaleParams{
		ID: id, TenantID: a.TenantID, LocationID: in.LocationID, UserID: a.UserID,
		CustomerID: in.CustomerID, CreatedAt: s.Now(),
	}
	if session != nil {
		sessionID := session.ID()
		params.CashSessionID = &sessionID
		span.SetAttributes(cashSessionAttr(sessionID))
	}
	sale, err := domain.NewSale(params)
	if err != nil {
		return nil, err
	}
	for _, l := range in.Lines {
		p := products[l.ProductID]
		err := sale.AddLine(domain.LineInput{
			ProductID: p.ID, ProductName: p.Name, Quantity: l.Quantity,
			UnitPrice: p.Price, TaxRate: p.TaxRate, Discount: l.Discount,
		}, limit)
		if errors.Is(err, domain.ErrDiscountExceedsLimit) {
			s.discountRejected.Add(ctx, 1)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := sale.Confirm(payments, s.Now()); err != nil {
		return nil, err
	}

	err = s.Sales.SaveConfirmed(ctx, sale, func(ctx context.Context) error {
		// The session may have closed since it was read: check it again,
		// locked, within the sale's transaction.
		if session != nil {
			if err := s.CashSessions.EnsureOpen(ctx, a.TenantID, session.ID()); err != nil {
				return err
			}
		}
		return s.Stock.ReserveSale(ctx, a.TenantID, a.UserID, sale.LocationID(), sale.ID(), stockLines(sale))
	})
	var stockErr *domain.InsufficientStockError
	if errors.As(err, &stockErr) {
		stockErr.ProductName = products[stockErr.ProductID].Name
	}
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int64("sale.number", sale.Number()))
	s.confirmed.Add(ctx, 1)
	s.amount.Record(ctx, sale.Totals().Total.InexactFloat64())
	return sale, nil
}

// products returns the cart's products by id, asking the catalog once for
// each. A product missing from the tenant's catalog is ErrProductNotFound;
// an inactive one, ErrProductInactive.
func (s *Service) products(ctx context.Context, tenantID uuid.UUID, lines []LineRequest) (map[uuid.UUID]ProductInfo, error) {
	var ids []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, l := range lines {
		if !seen[l.ProductID] {
			seen[l.ProductID] = true
			ids = append(ids, l.ProductID)
		}
	}
	found, err := s.Products.Products(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]ProductInfo, len(found))
	for _, p := range found {
		byID[p.ID] = p
	}
	for _, id := range ids {
		p, ok := byID[id]
		if !ok {
			return nil, domain.ErrProductNotFound
		}
		if !p.Active {
			return nil, domain.ErrProductInactive
		}
	}
	return byID, nil
}

// saleCashSession returns the actor's open cash session, which must be at
// locationID. Without one it returns nil if the tenant's policy allows it,
// and ErrCashSessionRequired otherwise.
func (s *Service) saleCashSession(ctx context.Context, a Actor, policy domain.SalesPolicy, locationID uuid.UUID) (*domain.CashSession, error) {
	session, err := s.CashSessions.Current(ctx, a.TenantID, a.UserID)
	switch {
	case errors.Is(err, domain.ErrCashSessionNotFound):
		if policy.CashSessionOptional {
			return nil, nil
		}
		s.cashSessionRequired.Add(ctx, 1)
		return nil, domain.ErrCashSessionRequired
	case err != nil:
		return nil, err
	case session.LocationID() != locationID:
		return nil, domain.ErrCashSessionLocationMismatch
	}
	return session, nil
}

func parsePayments(in []PaymentRequest) ([]domain.Payment, error) {
	payments := make([]domain.Payment, 0, len(in))
	for _, r := range in {
		m, err := domain.ParsePaymentMethod(r.Method)
		if err != nil {
			return nil, err
		}
		p, err := domain.NewPayment(m, r.Amount)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}

// stockLines adds up the sale's quantities by product, in the order each
// product first appears.
func stockLines(sale *domain.Sale) []StockLine {
	var out []StockLine
	index := map[uuid.UUID]int{}
	for _, l := range sale.Lines() {
		if i, ok := index[l.ProductID()]; ok {
			out[i].Quantity = out[i].Quantity.Add(l.Quantity())
			continue
		}
		index[l.ProductID()] = len(out)
		out = append(out, StockLine{ProductID: l.ProductID(), Quantity: l.Quantity()})
	}
	return out
}
