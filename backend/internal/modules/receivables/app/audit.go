package app

import (
	"context"
	"strconv"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

// auditCreditTermsChanged is the audit action of a changed credit limit or
// term of a customer (docs/auditoria.md).
const auditCreditTermsChanged = "credit.terms_changed"

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

// termsChange is the audit entry of a customer's terms going from old to
// next; ok is false when nothing changed.
func termsChange(customerID uuid.UUID, old, next domain.Terms) (_ AuditEntry, ok bool) {
	if old.Limit().Equal(next.Limit()) && old.Days() == next.Days() {
		return AuditEntry{}, false
	}
	return AuditEntry{Action: auditCreditTermsChanged, EntityID: customerID, Details: map[string]string{
		"old_limit": old.Limit().StringFixed(2), "new_limit": next.Limit().StringFixed(2),
		"old_term_days": strconv.Itoa(old.Days()), "new_term_days": strconv.Itoa(next.Days()),
	}}, true
}
