package app

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// cart is what a sale is built from: the tenant's policy, the cash session
// it is charged in (nil when the tenant lets users sell without one), the
// variants of the lines, the actor's discount limit, the payments and the
// unit and resolved price of each line, in order.
type cart struct {
	policy   domain.SalesPolicy
	session  *domain.CashSession
	variants map[uuid.UUID]VariantInfo
	limit    domain.DiscountLimit
	payments []domain.Payment
	units    []*UnitInfo
	prices   []ResolvedPrice
}

// checkCustomer checks that the sale's customer, if any, can buy: it is the
// tenant's and active. A sale without one is to the final consumer.
func (s *Service) checkCustomer(ctx context.Context, a Actor, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	return s.Customers.CheckCustomer(ctx, a.TenantID, a.UserID, *id)
}

// buildCart checks the input against the tenant's policy, cash session,
// customers and catalog, and resolves the price of each line.
func (s *Service) buildCart(ctx context.Context, a Actor, in ConfirmSaleInput) (cart, error) {
	if len(in.Lines) == 0 {
		return cart{}, domain.ErrSaleWithoutLines
	}
	var c cart
	var err error
	if c.policy, err = s.Policies.Get(ctx, a.TenantID); err != nil {
		return cart{}, err
	}
	session, ok, err := s.saleCashSession(ctx, a, c.policy, in.LocationID)
	if err != nil {
		return cart{}, err
	}
	if ok {
		c.session = session
	}
	if err := s.checkCustomer(ctx, a, in.CustomerID); err != nil {
		return cart{}, err
	}
	if c.variants, err = s.variants(ctx, a.TenantID, in.Lines); err != nil {
		return cart{}, err
	}
	c.limit = discountLimit(a, c.policy)
	if c.payments, err = parsePayments(in.Payments); err != nil {
		return cart{}, err
	}
	if c.units, err = lineUnits(in.Lines, c.variants); err != nil {
		return cart{}, err
	}
	if c.prices, err = s.resolvePrices(ctx, a, in, c.variants, c.units); err != nil {
		return cart{}, err
	}
	return c, nil
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
	ctx, span := s.spans.Start(ctx, "ResolvePrices", attribute.Int("sale.lines", len(in.Lines)))
	defer func() { usecase.End(span, err) }()

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
// locationID, and ok. Without one, ok is false if the tenant's policy allows
// it, and the error ErrCashSessionRequired otherwise.
func (s *Service) saleCashSession(ctx context.Context, a Actor, policy domain.SalesPolicy, locationID uuid.UUID) (_ *domain.CashSession, ok bool, err error) {
	session, err := s.CashSessions.Current(ctx, a.TenantID, a.UserID)
	switch {
	case errors.Is(err, domain.ErrCashSessionNotFound):
		if policy.CashSessionOptional {
			return nil, false, nil
		}
		s.cashSessionRequired.Add(ctx, 1)
		return nil, false, domain.ErrCashSessionRequired
	case err != nil:
		return nil, false, err
	case session.LocationID() != locationID:
		return nil, false, domain.ErrCashSessionLocationMismatch
	}
	return session, true, nil
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
