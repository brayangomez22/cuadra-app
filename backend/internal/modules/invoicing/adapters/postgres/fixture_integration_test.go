//go:build integration

package postgres_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through
// timestamptz. It is 2026-03-03 at noon in Bogotá.
var now = time.Date(2026, 3, 3, 17, 0, 0, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.SettingsRepository   = (*postgres.SettingsRepository)(nil)
	_ domain.ResolutionRepository = (*postgres.ResolutionRepository)(nil)
	_ domain.DocumentRepository   = (*postgres.DocumentRepository)(nil)
)

type fixture struct {
	db          *db.DB
	settings    *postgres.SettingsRepository
	resolutions *postgres.ResolutionRepository
	documents   *postgres.DocumentRepository
	queue       *queue
}

// queue records the issues enqueued, and fails them with err.
type queue struct {
	mu       sync.Mutex
	enqueued []uuid.UUID
	err      error
}

func (q *queue) enqueue(_ context.Context, _ pgx.Tx, _, documentID uuid.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.err != nil {
		return q.err
	}
	q.enqueued = append(q.enqueued, documentID)
	return nil
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	q := &queue{}
	return fixture{
		db: d, queue: q,
		settings:    postgres.NewSettingsRepository(d),
		resolutions: postgres.NewResolutionRepository(d),
		documents:   postgres.NewDocumentRepository(d, q.enqueue),
	}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// tenant is a tenant with a location, for the sales its documents reference.
type tenant struct {
	id, location uuid.UUID
	sales        int64
}

// createTenant stores a tenant and a location directly: invoicing depends
// on neither identity nor inventory, but its tables reference them.
func (f fixture) createTenant(t *testing.T) *tenant {
	t.Helper()
	tn := &tenant{id: newID(), location: newID()}
	f.exec(t, tn.id, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2)", tn.id, now)
	f.exec(t, tn.id, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, 'Principal', true, $3)", tn.location, tn.id, now)
	return tn
}

// createSale stores a confirmed sale, for a document to reference.
func (f fixture) createSale(t *testing.T, tn *tenant) uuid.UUID {
	t.Helper()
	id := newID()
	tn.sales++
	f.exec(t, tn.id, `INSERT INTO sales (id, tenant_id, location_id, number, user_id, status, subtotal, discounts, tax,
		total, paid, change, created_at, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, 'confirmed', 100, 0, 0, 100, 100, 0, $6, $6)`, id, tn.id, tn.location, tn.sales, newID(), now)
	return id
}

func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// createResolution stores a resolution of kind and prefix from 1 to to,
// valid all of 2026.
func (f fixture) createResolution(t *testing.T, tn *tenant, kind domain.Kind, prefix string, to int64) *domain.Resolution {
	t.Helper()
	r, err := domain.NewResolution(newID(), tn.id, domain.ResolutionParams{
		Kind: kind, Prefix: prefix, Number: "18764000001234", From: 1, To: to,
		ValidFrom: domain.NewDate(2026, time.January, 1), ValidTo: domain.NewDate(2026, time.December, 31),
	}, now)
	require.NoError(t, err)
	require.NoError(t, f.resolutions.Create(t.Context(), r))
	return r
}

// register numbers a new sale's POS document in one transaction, as the
// sale's confirmation does.
func (f fixture) register(t *testing.T, tn *tenant) (*domain.Document, error) {
	t.Helper()
	return f.registerSale(t, tn, f.createSale(t, tn))
}

// registerSale numbers the POS document of saleID; it may run concurrently.
func (f fixture) registerSale(t *testing.T, tn *tenant, saleID uuid.UUID) (*domain.Document, error) {
	t.Helper()
	var doc *domain.Document
	err := f.db.InTenantTx(t.Context(), tn.id, func(ctx context.Context, _ pgx.Tx) error {
		rs, err := f.resolutions.LockKind(ctx, tn.id, domain.KindPOS)
		if err != nil {
			return err
		}
		r, err := domain.PickResolution(rs, domain.KindPOS, now)
		if err != nil {
			return err
		}
		n, err := r.Draw(now)
		if err != nil {
			return err
		}
		if err := f.resolutions.SaveNext(ctx, r); err != nil {
			return err
		}
		doc, err = domain.NewDocument(domain.DocumentParams{
			ID: newID(), TenantID: tn.id, SaleID: saleID, CreatedBy: newID(),
			Kind: domain.KindPOS, Resolution: r, Number: n, IssuedAt: now,
		})
		if err != nil {
			return err
		}
		return f.documents.Add(ctx, doc)
	})
	return doc, err
}
