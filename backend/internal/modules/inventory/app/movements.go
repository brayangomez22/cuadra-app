package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// ReceiptInput is an entry of goods at a location. It names the variant, or
// (deprecated) the product whose default variant receives it; not both.
type ReceiptInput struct {
	VariantID uuid.UUID
	// ProductID, deprecated in the contract, resolves to the product's default
	// variant. T24j removes it.
	ProductID  uuid.UUID
	LocationID uuid.UUID
	Quantity   decimal.Decimal
	UnitCost   decimal.Decimal
}

// AdjustmentInput is a correction of the stock at a location; Quantity is
// signed. It names the variant or the product like ReceiptInput.
type AdjustmentInput struct {
	VariantID uuid.UUID
	// ProductID, deprecated in the contract, resolves to the product's default
	// variant. T24j removes it.
	ProductID  uuid.UUID
	LocationID uuid.UUID
	Quantity   decimal.Decimal
	Reason     string
}

// TransferInput moves stock of a variant between two locations. It names
// the variant or the product like ReceiptInput.
type TransferInput struct {
	VariantID uuid.UUID
	// ProductID, deprecated in the contract, resolves to the product's default
	// variant. T24j removes it.
	ProductID      uuid.UUID
	FromLocationID uuid.UUID
	ToLocationID   uuid.UUID
	Quantity       decimal.Decimal
}

// ReceiveStock registers an entry of goods (purchase_in) and updates the
// weighted average cost.
func (s *Service) ReceiveStock(ctx context.Context, a Actor, in ReceiptInput) (_ *domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "ReceiveStock", a, locationAttr(in.LocationID))
	defer func() { endSpan(span, err) }()

	if in.VariantID, err = s.resolveVariant(ctx, a, span, in.VariantID, in.ProductID); err != nil {
		return nil, err
	}
	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return nil, err
	}
	var m *domain.StockMovement
	err = s.Stock.Apply(ctx, a.TenantID, in.VariantID, []uuid.UUID{in.LocationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			var err error
			m, err = levels[0].Receive(in.Quantity, in.UnitCost, s.info(a))
			return []*domain.StockMovement{m}, err
		})
	if err != nil {
		return nil, err
	}
	s.countMovements(ctx, m)
	return m, nil
}

// AdjustStock registers a correction of the stock (adjustment). A negative
// one follows the tenant's stock policy.
func (s *Service) AdjustStock(ctx context.Context, a Actor, in AdjustmentInput) (_ *domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "AdjustStock", a, locationAttr(in.LocationID))
	defer func() { endSpan(span, err) }()

	if in.VariantID, err = s.resolveVariant(ctx, a, span, in.VariantID, in.ProductID); err != nil {
		return nil, err
	}
	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return nil, err
	}
	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	var m *domain.StockMovement
	err = s.Stock.Apply(ctx, a.TenantID, in.VariantID, []uuid.UUID{in.LocationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			var err error
			m, err = levels[0].Adjust(in.Quantity, in.Reason, policy, s.info(a))
			return []*domain.StockMovement{m}, err
		})
	if err != nil {
		s.countInsufficient(ctx, err, domain.MovementAdjustment)
		return nil, err
	}
	s.countMovements(ctx, m)
	return m, nil
}

// TransferStock moves stock between two locations: a transfer_out at the
// origin and a transfer_in at the destination, stored together.
func (s *Service) TransferStock(ctx context.Context, a Actor, in TransferInput) (out, inMov *domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "TransferStock", a,
		attribute.String("location.from_id", in.FromLocationID.String()),
		attribute.String("location.to_id", in.ToLocationID.String()))
	defer func() { endSpan(span, err) }()

	if in.VariantID, err = s.resolveVariant(ctx, a, span, in.VariantID, in.ProductID); err != nil {
		return nil, nil, err
	}
	// domain.Transfer checks it too; checking first keeps Apply from being
	// asked to lock the same level twice.
	if in.FromLocationID == in.ToLocationID {
		return nil, nil, domain.ErrSameLocationTransfer
	}
	for _, id := range []uuid.UUID{in.FromLocationID, in.ToLocationID} {
		if err := s.requireActiveLocation(ctx, a, id); err != nil {
			return nil, nil, err
		}
	}
	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return nil, nil, err
	}
	err = s.Stock.Apply(ctx, a.TenantID, in.VariantID, []uuid.UUID{in.FromLocationID, in.ToLocationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			var err error
			out, inMov, err = domain.Transfer(levels[0], levels[1], in.Quantity, policy, a.UserID, s.Now())
			return []*domain.StockMovement{out, inMov}, err
		})
	if err != nil {
		s.countInsufficient(ctx, err, domain.MovementTransferOut)
		return nil, nil, err
	}
	s.countMovements(ctx, out, inMov)
	return out, inMov, nil
}

// resolveVariant returns the variant a movement is for: variantID, or the
// default variant of productID (deprecated). Exactly one of them must be set;
// otherwise ErrVariantRequired. It records the variant on the span.
func (s *Service) resolveVariant(ctx context.Context, a Actor, span trace.Span, variantID, productID uuid.UUID) (uuid.UUID, error) {
	switch {
	case (variantID == uuid.Nil) == (productID == uuid.Nil):
		return uuid.Nil, domain.ErrVariantRequired
	case productID != uuid.Nil:
		var err error
		if variantID, err = s.Variants.DefaultVariant(ctx, a.TenantID, productID); err != nil {
			return uuid.Nil, err
		}
	}
	span.SetAttributes(variantAttr(variantID))
	return variantID, nil
}

// info is who registers a movement and when: the actor, now.
func (s *Service) info(a Actor) domain.MovementInfo {
	return domain.MovementInfo{UserID: a.UserID, OccurredAt: s.Now()}
}

func (s *Service) countMovements(ctx context.Context, movements ...*domain.StockMovement) {
	for _, m := range movements {
		s.movements.Add(ctx, 1, metric.WithAttributes(attribute.String("type", m.Type().String())))
	}
}

// countInsufficient counts err if it is a rejection for lack of stock of an
// exit of type t.
func (s *Service) countInsufficient(ctx context.Context, err error, t domain.MovementType) {
	if errors.Is(err, domain.ErrInsufficientStock) {
		s.insufficientStock.Add(ctx, 1, metric.WithAttributes(attribute.String("type", t.String())))
	}
}
