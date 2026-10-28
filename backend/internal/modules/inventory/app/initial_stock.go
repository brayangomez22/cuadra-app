package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// InitialStockLine is the initial stock of a variant, in its base unit, at
// its unit cost.
type InitialStockLine struct {
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
}

// ReceiveInitialStock registers the initial stock of variants at a
// location, one initial entry (initial_in) per line, as an import loads
// it. With a ctx from db.InTenantTx it joins the import's batch, so the
// entries are written with the products they belong to.
func (s *Service) ReceiveInitialStock(ctx context.Context, a Actor, locationID uuid.UUID, lines []InitialStockLine) (err error) {
	ctx, span := s.startSpan(ctx, "ReceiveInitialStock", a, locationAttr(locationID), attribute.Int("lines", len(lines)))
	defer func() { usecase.End(span, err) }()

	if err := s.requireActiveLocation(ctx, a, locationID); err != nil {
		return err
	}
	movements := make([]*domain.StockMovement, 0, len(lines))
	for _, l := range lines {
		err := s.Stock.Apply(ctx, a.TenantID, l.VariantID, []uuid.UUID{locationID},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].ReceiveInitial(l.Quantity, l.UnitCost, s.info(a))
				if err != nil {
					return nil, err
				}
				movements = append(movements, m)
				return []*domain.StockMovement{m}, nil
			})
		if err != nil {
			return err
		}
	}
	s.countMovements(ctx, movements...)
	return nil
}

// CheckLocation fails unless id is an active location of the actor's
// tenant (domain.ErrLocationNotFound, domain.ErrLocationInactive). It is
// for other modules: an import checks its location before reading a file.
func (s *Service) CheckLocation(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "CheckLocation", a, locationAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.requireActiveLocation(ctx, a, id)
}
