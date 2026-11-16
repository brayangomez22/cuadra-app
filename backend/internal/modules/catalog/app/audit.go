package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// auditVariantPriceChanged is the audit action of a new price (docs/auditoria.md).
const auditVariantPriceChanged = "variant.price_changed"

// AuditEntry is what an audited operation of the module did: an action of
// the catalog, the entity it is about and its details (amounts as decimal
// strings).
type AuditEntry struct {
	Action   string
	EntityID uuid.UUID
	Details  map[string]string
}

// AuditLog records the module's audited operations in the audit module.
// It is called with the ctx of the operation's transaction: its error must
// undo the operation.
type AuditLog interface {
	Record(ctx context.Context, tenantID, userID uuid.UUID, e AuditEntry) error
}

// Transactor runs fn in one transaction of tenantID; every repository and
// port called with fn's ctx joins it.
type Transactor interface {
	InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error
}

// variantPrices returns the price of each variant of p, to compare after a
// change.
func variantPrices(p *domain.Product) map[uuid.UUID]decimal.Decimal {
	out := map[uuid.UUID]decimal.Decimal{}
	for _, v := range p.Variants() {
		out[v.ID()] = v.Price()
	}
	return out
}

// saveProduct stores p and, in the same transaction, an audit event for
// each variant whose price differs from before (variants added since have
// no event: a new price is not a change). It returns how many changed.
func (s *Service) saveProduct(ctx context.Context, a Actor, p *domain.Product, before map[uuid.UUID]decimal.Decimal) (int, error) {
	changed := 0
	err := s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		if err := s.Products.Update(ctx, p); err != nil {
			return err
		}
		for _, v := range p.Variants() {
			old, ok := before[v.ID()]
			if !ok || old.Equal(v.Price()) {
				continue
			}
			changed++
			err := s.Audit.Record(ctx, a.TenantID, a.UserID, AuditEntry{
				Action: auditVariantPriceChanged, EntityID: v.ID(),
				Details: map[string]string{"old_price": old.StringFixed(2), "new_price": v.Price().StringFixed(2)},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return changed, err
}
