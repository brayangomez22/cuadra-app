package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// ReceiptInput is an entry of goods of a variant at a location.
type ReceiptInput struct {
	VariantID  uuid.UUID
	LocationID uuid.UUID
	// Quantity and UnitCost are in UnitID, or in the product's base unit
	// when UnitID is nil.
	Quantity decimal.Decimal
	UnitCost decimal.Decimal
	// UnitID is an alternate unit of the variant (a unit of the tenant), or
	// nil for the base unit. The movement is stored in the base unit.
	UnitID *uuid.UUID
}

// AdjustmentInput is a correction of a variant's stock at a location;
// Quantity is signed.
type AdjustmentInput struct {
	VariantID  uuid.UUID
	LocationID uuid.UUID
	Quantity   decimal.Decimal
	Reason     string
}

// TransferInput moves stock of a variant between two locations.
type TransferInput struct {
	VariantID      uuid.UUID
	FromLocationID uuid.UUID
	ToLocationID   uuid.UUID
	Quantity       decimal.Decimal
}

// ReceiveStock registers an entry of goods (purchase_in) and updates the
// weighted average cost. An entry in an alternate unit is converted to the
// base unit first.
func (s *Service) ReceiveStock(ctx context.Context, a Actor, in ReceiptInput) (_ *domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "ReceiveStock", a, locationAttr(in.LocationID), variantAttr(in.VariantID),
		attribute.Bool("unit.alternate", in.UnitID != nil))
	defer func() { endSpan(span, err) }()

	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return nil, err
	}
	if in.UnitID != nil {
		factor, err := s.Variants.UnitFactor(ctx, a.TenantID, in.VariantID, *in.UnitID)
		if err != nil {
			return nil, err
		}
		if in.Quantity, in.UnitCost, err = domain.ToBaseUnit(in.Quantity, in.UnitCost, factor); err != nil {
			return nil, err
		}
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
	ctx, span := s.startSpan(ctx, "AdjustStock", a, locationAttr(in.LocationID), variantAttr(in.VariantID))
	defer func() { endSpan(span, err) }()

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
		attribute.String("location.to_id", in.ToLocationID.String()),
		variantAttr(in.VariantID))
	defer func() { endSpan(span, err) }()

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
