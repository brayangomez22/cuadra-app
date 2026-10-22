package app

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
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
// optional.
type ConfirmSaleInput struct {
	LocationID uuid.UUID
	CustomerID *uuid.UUID
	Lines      []LineRequest
	Payments   []PaymentRequest
}

// ConfirmSale builds the sale from the cart with the prices of the
// PriceResolver, checks the discounts against the actor's limit and the
// payments against the total, and stores it together with its stock exits
// (by variant) and its consecutive number. Prices are resolved before the
// transaction that draws the number, so a slow rule does not hold the
// location's sequence. The sale is charged in the actor's open cash session,
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
	variants, err := s.variants(ctx, a.TenantID, in.Lines)
	if err != nil {
		return nil, err
	}
	limit := discountLimit(a, policy)
	payments, err := parsePayments(in.Payments)
	if err != nil {
		return nil, err
	}
	units, err := lineUnits(in.Lines, variants)
	if err != nil {
		return nil, err
	}
	prices, err := s.resolvePrices(ctx, a, in, variants, units)
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
	alternate := 0
	for i, l := range in.Lines {
		v := variants[l.VariantID]
		unit := domain.BaseUnit
		if u := units[i]; u != nil {
			unit = domain.LineUnit{Code: u.Code, Factor: u.Factor}
			alternate++
		}
		err := sale.AddLine(domain.LineInput{
			VariantID: v.ID, ProductID: v.ProductID, ProductName: v.ProductName, VariantLabel: v.Label,
			Quantity: l.Quantity, Unit: unit, UnitPrice: prices[i].UnitPrice, PriceRule: prices[i].Rule,
			TaxRate: v.TaxRate, Discount: l.Discount,
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
		v := variants[stockErr.VariantID]
		stockErr.ProductName, stockErr.VariantLabel = v.ProductName, v.Label
		span.SetAttributes(attribute.String("variant.id", stockErr.VariantID.String()))
	}
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int64("sale.number", sale.Number()))
	s.confirmed.Add(ctx, 1)
	s.alternateUnitLines.Add(ctx, int64(alternate))
	s.amount.Record(ctx, sale.Totals().Total.InexactFloat64())
	for _, p := range prices {
		s.priceRules.Add(ctx, 1, metric.WithAttributes(attribute.String("price.rule", string(p.Rule))))
	}
	return sale, nil
}

// variants returns the cart's variants by id, asking the catalog once for
// each. A variant missing from the tenant's catalog is ErrProductNotFound;
// one that cannot be sold, because it or its product is inactive,
// ErrProductInactive.
func (s *Service) variants(ctx context.Context, tenantID uuid.UUID, lines []LineRequest) (map[uuid.UUID]VariantInfo, error) {
	var ids []uuid.UUID
	for _, l := range lines {
		if !slices.Contains(ids, l.VariantID) {
			ids = append(ids, l.VariantID)
		}
	}
	found, err := s.Catalog.Variants(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]VariantInfo, len(found))
	for _, v := range found {
		byID[v.ID] = v
	}
	for _, id := range ids {
		v, ok := byID[id]
		if !ok {
			return nil, domain.ErrProductNotFound
		}
		if !v.Active {
			return nil, domain.ErrProductInactive
		}
	}
	return byID, nil
}

// lineUnits returns the alternate unit of each line of the cart, in order,
// or nil for a line in the base unit. A unit the variant does not have is
// ErrUnitNotAvailable.
func lineUnits(lines []LineRequest, variants map[uuid.UUID]VariantInfo) ([]*UnitInfo, error) {
	units := make([]*UnitInfo, len(lines))
	for i, l := range lines {
		if l.UnitID == nil {
			continue
		}
		v := variants[l.VariantID]
		j := slices.IndexFunc(v.Units, func(u UnitInfo) bool { return u.UnitID == *l.UnitID })
		if j < 0 {
			return nil, domain.ErrUnitNotAvailable
		}
		units[i] = &v.Units[j]
	}
	return units, nil
}

// resolvePrices asks the PriceResolver the unit price of each line of the
// cart, in order: the price of one unit of the line, base or alternate.
func (s *Service) resolvePrices(ctx context.Context, a Actor, in ConfirmSaleInput, variants map[uuid.UUID]VariantInfo, units []*UnitInfo) (_ []ResolvedPrice, err error) {
	ctx, span := s.tracer.Start(ctx, "sales.ResolvePrices", trace.WithAttributes(attribute.Int("sale.lines", len(in.Lines))))
	defer func() { endSpan(span, err) }()

	prices := make([]ResolvedPrice, len(in.Lines))
	for i, l := range in.Lines {
		v := variants[l.VariantID]
		q := PriceQuery{
			TenantID: a.TenantID, LocationID: in.LocationID, CustomerID: in.CustomerID,
			VariantID: v.ID, ProductID: v.ProductID, BasePrice: v.Price, Quantity: l.Quantity,
			UnitFactor: domain.BaseUnit.Factor,
		}
		if u := units[i]; u != nil {
			q.UnitID, q.UnitFactor, q.BasePrice = &u.UnitID, u.Factor, u.Price
		}
		prices[i], err = s.Prices.Resolve(ctx, q)
		if err != nil {
			return nil, err
		}
	}
	return prices, nil
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
