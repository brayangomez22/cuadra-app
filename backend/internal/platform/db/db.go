// Package db provides the PostgreSQL connection pool and the transaction
// helpers every repository runs through.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ErrNoTenant is returned by WithTenantTx when called without a tenant.
var ErrNoTenant = errors.New("db: tenant id is required")

// ErrTenantMismatch is returned by WithTenantTx when the context carries a
// transaction of another tenant.
var ErrTenantMismatch = errors.New("db: transaction belongs to another tenant")

// ErrUnsafeRole is returned by Open when the connection role could bypass
// Row-Level Security: a superuser, a BYPASSRLS role or the owner of a table.
var ErrUnsafeRole = errors.New("db: connection role can bypass row-level security")

// unsafeRoleQuery reports whether the current role escapes RLS. Owners are
// exempt from it unless the table uses FORCE ROW LEVEL SECURITY, so the app
// must never own a table.
const unsafeRoleQuery = `
SELECT r.rolsuper OR r.rolbypassrls OR EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relowner = r.oid
)
FROM pg_roles r
WHERE r.rolname = current_user`

// DB is the application's connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to url with query spans and pool metrics, checks that the
// database answers and refuses a role that could bypass Row-Level Security.
func Open(ctx context.Context, url string, tp trace.TracerProvider, mp metric.MeterProvider) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// The error may echo the URL, password included.
		return nil, errors.New("db: invalid database url")
	}
	// Query parameters are left out of spans: they carry tenant ids and user data.
	cfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tp),
		otelpgx.WithMeterProvider(mp),
		otelpgx.WithSpanNameCtxFunc(querySpanName),
	)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}
	d := &DB{pool: pool}

	if err := d.checkRole(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(mp)); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: record pool stats: %w", err)
	}
	return d, nil
}

func (d *DB) checkRole(ctx context.Context) error {
	var unsafe bool
	if err := d.pool.QueryRow(ctx, unsafeRoleQuery).Scan(&unsafe); err != nil {
		return fmt.Errorf("db: check connection role: %w", err)
	}
	if unsafe {
		return ErrUnsafeRole
	}
	return nil
}

// Close closes the pool.
func (d *DB) Close() { d.pool.Close() }

// Pool returns the underlying pool for infrastructure that needs it, such as
// the job queue. Business code goes through WithTenantTx instead.
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

// Ping checks that the database answers.
func (d *DB) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }

// WithTx runs fn in a transaction. It commits when fn returns nil and rolls
// back when fn returns an error (which is returned as is) or panics (the panic
// keeps propagating). Business tables are queried only through WithTenantTx.
func (d *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, d.pool, fn)
}

// WithTenantTx runs fn in a transaction scoped to tenantID: Row-Level Security
// policies read it from app.tenant_id. The setting is local to the
// transaction, so it never leaks to the next use of the pooled connection.
//
// If ctx carries a transaction opened by InTenantTx, fn joins it as a
// savepoint instead: its writes commit or roll back with that transaction,
// and an error of fn undoes only what fn wrote. Joining a transaction of
// another tenant fails with ErrTenantMismatch.
func (d *DB) WithTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	if tenantID == uuid.Nil {
		return ErrNoTenant
	}
	if amb, ok := ctx.Value(ambientKey{}).(ambient); ok {
		if amb.tenantID != tenantID {
			return ErrTenantMismatch
		}
		return pgx.BeginFunc(ctx, amb.tx, fn)
	}
	return d.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
			return fmt.Errorf("db: set tenant: %w", err)
		}
		return fn(tx)
	})
}

// ambientKey is the context key of the transaction InTenantTx shares.
type ambientKey struct{}

type ambient struct {
	tenantID uuid.UUID
	tx       pgx.Tx
}

// InTenantTx runs fn in a transaction scoped to tenantID, like WithTenantTx,
// and passes it a context that carries the transaction: WithTenantTx calls
// made with that context, from any module, join it. This is how one use case
// stores the changes of several modules atomically without the transaction
// reaching its code. If ctx already carries a transaction, fn joins it too.
func (d *DB) InTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context, pgx.Tx) error) error {
	return d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, ambientKey{}, ambient{tenantID: tenantID, tx: tx}), tx)
	})
}
