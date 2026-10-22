package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// VariantUnitInput is the data of an alternate unit of a variant: how many
// base units it holds, its price without IVA and an optional barcode.
type VariantUnitInput struct {
	Factor  decimal.Decimal
	Price   decimal.Decimal
	Barcode string
}

// SetVariantUnit lets a variant be sold and bought in unitID, a unit of the
// tenant, and returns the product. If the variant already has that unit, it
// replaces its data. The stock stays in the product's base unit.
func (s *Service) SetVariantUnit(ctx context.Context, a Actor, productID, variantID, unitID uuid.UUID, in VariantUnitInput) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "SetVariantUnit", a, productAttr(productID), variantAttr(variantID), unitAttr(unitID))
	defer func() { endSpan(span, err) }()

	p, err := s.Products.GetByID(ctx, a.TenantID, productID)
	if err != nil {
		return nil, err
	}
	unit, err := s.Units.GetByID(ctx, a.TenantID, unitID)
	if err != nil {
		return nil, err
	}
	if err := p.SetVariantUnit(variantID, domain.VariantUnitParams{
		Unit: unit.Ref(), Factor: in.Factor, Price: in.Price, Barcode: in.Barcode,
	}); err != nil {
		return nil, err
	}
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// RemoveVariantUnit stops selling and buying a variant in unitID and
// returns the product. Past sales keep the unit they were sold in.
func (s *Service) RemoveVariantUnit(ctx context.Context, a Actor, productID, variantID, unitID uuid.UUID) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "RemoveVariantUnit", a, productAttr(productID), variantAttr(variantID), unitAttr(unitID))
	defer func() { endSpan(span, err) }()

	p, err := s.Products.GetByID(ctx, a.TenantID, productID)
	if err != nil {
		return nil, err
	}
	if err := p.RemoveVariantUnit(variantID, unitID); err != nil {
		return nil, err
	}
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
