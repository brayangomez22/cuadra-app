package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// auditTenantSettingsChanged is the audit action of a changed setting of
// the company (docs/auditoria.md): one event per setting.
const auditTenantSettingsChanged = "tenant.settings_changed"

// AuditEntry is what an audited operation of the module did: an action of
// the catalog, the entity it is about and its details.
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

// setting is a setting of the company as the audit log names and shows it.
type setting struct{ name, value string }

// tenantSettings returns the audited settings of t, by name.
func tenantSettings(t *domain.Tenant) []setting {
	c := t.Contact()
	return []setting{
		{"address", c.Address()},
		{"city", c.City()},
		{"default_tax_rate", t.DefaultTaxRate().String()},
		{"municipality_code", c.MunicipalityCode()},
		{"phone", c.Phone()},
	}
}

// auditSettings records one event per setting of the tenant that differs
// from before.
func (s *Service) auditSettings(ctx context.Context, userID uuid.UUID, t *domain.Tenant, before []setting) error {
	for i, after := range tenantSettings(t) {
		if after.value == before[i].value {
			continue
		}
		err := s.Audit.Record(ctx, t.ID(), userID, AuditEntry{
			Action: auditTenantSettingsChanged, EntityID: t.ID(),
			Details: map[string]string{"setting": after.name, "old": before[i].value, "new": after.value},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
