//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 11, 15, 20, 30, 15, 123456000, time.UTC)

// Compile-time check: the adapter satisfies the domain port.
var _ domain.EventRepository = (*postgres.EventRepository)(nil)

type fixture struct {
	db     *db.DB
	events *postgres.EventRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, events: postgres.NewEventRepository(d)}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// createTenant stores a tenant row directly: audit does not depend on the
// identity module, but its table references tenants.
func (f fixture) createTenant(t *testing.T, demo bool) uuid.UUID {
	t.Helper()
	id := newID()
	require.NoError(t, f.db.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at, demo) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2, $3)",
			id, now, demo)
		return err
	}))
	return id
}

// priceChanged is an event of tenantID: a variant's price went from old to
// next.
func priceChanged(t *testing.T, tenantID, userID uuid.UUID, at time.Time, old, next string) *domain.Event {
	t.Helper()
	variant := newID()
	e, err := domain.NewEvent(domain.EventInput{
		ID: newID(), TenantID: tenantID, UserID: userID, Role: "admin",
		Action: domain.ActionVariantPriceChanged, EntityID: &variant, OccurredAt: at,
		Details: map[string]string{"old_price": old, "new_price": next},
	})
	require.NoError(t, err)
	return e
}

// moduleEnabled is an event of tenantID without entity id.
func moduleEnabled(t *testing.T, tenantID, userID uuid.UUID, at time.Time) *domain.Event {
	t.Helper()
	e, err := domain.NewEvent(domain.EventInput{
		ID: newID(), TenantID: tenantID, UserID: userID, Role: "owner",
		Action: domain.ActionModuleEnabled, OccurredAt: at, Details: map[string]string{"module": "variants"},
	})
	require.NoError(t, err)
	return e
}

func (f fixture) list(t *testing.T, filter domain.EventFilter) domain.EventPage {
	t.Helper()
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	page, err := f.events.List(t.Context(), filter)
	require.NoError(t, err)
	return page
}
