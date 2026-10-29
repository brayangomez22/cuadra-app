package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// LineRequest is a line of the cart: a variant, how much and the discount
// (an amount of COP before IVA). The price comes from the PriceResolver and
// the IVA from the catalog, never from the request.
type LineRequest struct {
	VariantID uuid.UUID
	// Quantity is in UnitID, or in the product's base unit when UnitID is
	// nil.
	Quantity decimal.Decimal
	Discount decimal.Decimal
	// UnitID is an alternate unit of the variant (a unit of the tenant), or
	// nil for the base unit.
	UnitID *uuid.UUID
}

// PaymentRequest is a payment of the cart; Method is a domain.PaymentMethod
// code.
type PaymentRequest struct {
	Method string
	Amount decimal.Decimal
}

// ConfirmSaleInput is a whole cart, checked out at once. CustomerID is
// optional. CreditAuthorization, also optional, are the credentials of who
// lets a credit sale exceed the customer's available credit.
type ConfirmSaleInput struct {
	LocationID          uuid.UUID
	CustomerID          *uuid.UUID
	Lines               []LineRequest
	Payments            []PaymentRequest
	CreditAuthorization *Credentials
}

// ConfirmSale builds the sale from the cart with the prices of the
// PriceResolver, checks the discounts against the actor's limit and the
// payments against the total, and stores it together with its stock exits
// (by variant) and its consecutive number. Prices are resolved before the
// transaction that draws the number, so a slow rule does not hold the
// location's sequence. The sale is charged in the actor's open cash session,
// which must be at the sale's location; without one, the sale is rejected
// unless the tenant lets users sell without it. The part paid on credit
// opens a receivable of the customer in the same transaction; who authorizes
// exceeding the customer's credit is checked before it, since checking a
// password is slow. If any step fails, nothing is stored.
func (s *Service) ConfirmSale(ctx context.Context, a Actor, in ConfirmSaleInput) (_ *domain.Sale, err error) {
	ctx, span := s.startSpan(ctx, "ConfirmSale", a, attribute.String("location.id", in.LocationID.String()))
	defer func() { usecase.End(span, err) }()

	c, err := s.buildCart(ctx, a, in)
	if err != nil {
		return nil, err
	}
	sale, err := s.newSale(ctx, span, a, in, c)
	if err != nil {
		return nil, err
	}
	if c.authorizedBy, err = s.authorizeCredit(ctx, a, sale, in.CreditAuthorization); err != nil {
		return nil, err
	}
	if err := s.saveSale(ctx, span, a, sale, c); err != nil {
		return nil, err
	}
	s.recordSale(ctx, span, sale, c)
	return sale, nil
}

// newSale builds the confirmed sale of the cart, still without a number.
func (s *Service) newSale(ctx context.Context, span trace.Span, a Actor, in ConfirmSaleInput, c cart) (*domain.Sale, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	span.SetAttributes(saleAttr(id))
	params := domain.NewSaleParams{
		ID: id, TenantID: a.TenantID, LocationID: in.LocationID, UserID: a.UserID,
		CustomerID: in.CustomerID, CreatedAt: s.Now(),
	}
	if c.session != nil {
		sessionID := c.session.ID()
		params.CashSessionID = &sessionID
		span.SetAttributes(cashSessionAttr(sessionID))
	}
	sale, err := domain.NewSale(params)
	if err != nil {
		return nil, err
	}
	if err := s.addLines(ctx, sale, in, c); err != nil {
		return nil, err
	}
	if err := sale.Confirm(c.payments, s.Now()); err != nil {
		return nil, err
	}
	return sale, nil
}

// addLines adds the cart's lines to the sale, each in its unit and at its
// resolved price.
func (s *Service) addLines(ctx context.Context, sale *domain.Sale, in ConfirmSaleInput, c cart) error {
	for i, l := range in.Lines {
		v := c.variants[l.VariantID]
		unit := domain.BaseUnit
		if u := c.units[i]; u != nil {
			unit = domain.LineUnit{Code: u.Code, Factor: u.Factor}
		}
		err := sale.AddLine(domain.LineInput{
			VariantID: v.ID, ProductID: v.ProductID, ProductName: v.ProductName, VariantLabel: v.Label,
			Quantity: l.Quantity, Unit: unit, UnitPrice: c.prices[i].UnitPrice, PriceRule: c.prices[i].Rule,
			TaxRate: v.TaxRate, Discount: l.Discount,
		}, c.limit)
		if errors.Is(err, domain.ErrDiscountExceedsLimit) {
			s.discountRejected.Add(ctx, 1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// saveSale stores the sale with its number and takes its stock, in one
// transaction. A missing stock names the variant that lacks it.
func (s *Service) saveSale(ctx context.Context, span trace.Span, a Actor, sale *domain.Sale, c cart) error {
	err := s.Sales.SaveConfirmed(ctx, sale, func(ctx context.Context) error {
		// The session may have closed since it was read: check it again,
		// locked, within the sale's transaction.
		if c.session != nil {
			if err := s.CashSessions.EnsureOpen(ctx, a.TenantID, c.session.ID()); err != nil {
				return err
			}
		}
		if err := s.Stock.ReserveSale(ctx, a.TenantID, a.UserID, sale.LocationID(), sale.ID(), stockLines(sale)); err != nil {
			return err
		}
		return s.chargeCredit(ctx, a, sale, c.authorizedBy)
	})
	var stockErr *domain.InsufficientStockError
	if errors.As(err, &stockErr) {
		v := c.variants[stockErr.VariantID]
		stockErr.ProductName, stockErr.VariantLabel = v.ProductName, v.Label
		span.SetAttributes(attribute.String("variant.id", stockErr.VariantID.String()))
	}
	return err
}

// recordSale counts a confirmed sale in its metrics.
func (s *Service) recordSale(ctx context.Context, span trace.Span, sale *domain.Sale, c cart) {
	alternate := 0
	for _, u := range c.units {
		if u != nil {
			alternate++
		}
	}
	span.SetAttributes(attribute.Int64("sale.number", sale.Number()))
	s.confirmed.Add(ctx, 1)
	s.alternateUnitLines.Add(ctx, int64(alternate))
	s.amount.Record(ctx, sale.Totals().Total.InexactFloat64())
	for _, p := range c.prices {
		s.priceRules.Add(ctx, 1, metric.WithAttributes(attribute.String("price.rule", string(p.Rule))))
	}
}

// stockLines adds up the sale's quantities by variant, in the base unit, in
// the order each variant first appears.
func stockLines(sale *domain.Sale) []StockLine {
	var out []StockLine
	index := map[uuid.UUID]int{}
	for _, l := range sale.Lines() {
		if i, ok := index[l.VariantID()]; ok {
			out[i].Quantity = out[i].Quantity.Add(l.BaseQuantity())
			continue
		}
		index[l.VariantID()] = len(out)
		out = append(out, StockLine{VariantID: l.VariantID(), Quantity: l.BaseQuantity()})
	}
	return out
}
