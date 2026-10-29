package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// SupplierRepository implements domain.SupplierRepository.
type SupplierRepository struct {
	db *db.DB
}

// NewSupplierRepository returns a SupplierRepository on d.
func NewSupplierRepository(d *db.DB) *SupplierRepository { return &SupplierRepository{db: d} }

// Create stores a new supplier.
func (r *SupplierRepository) Create(ctx context.Context, s *domain.Supplier) error {
	snap := s.Snapshot()
	days, err := termDays(snap.PaymentTermDays)
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, snap.TenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateSupplier(ctx, sqlcgen.CreateSupplierParams{
			ID: snap.ID, TenantID: snap.TenantID, Name: snap.Name, Nit: snap.NIT,
			ContactName: nullable(snap.Contact.Name), Email: nullable(snap.Contact.Email),
			Phone: nullable(snap.Contact.Phone), PaymentTermDays: days,
			Active: snap.Active, CreatedAt: snap.CreatedAt,
		})
	})
	return supplierWriteError("create", err)
}

// Update stores the supplier's data; updated_at is the database's clock.
func (r *SupplierRepository) Update(ctx context.Context, s *domain.Supplier) error {
	snap := s.Snapshot()
	days, err := termDays(snap.PaymentTermDays)
	if err != nil {
		return err
	}
	affected, err := db.Affected(ctx, r.db, snap.TenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).UpdateSupplier(ctx, sqlcgen.UpdateSupplierParams{
			ID: snap.ID, Name: snap.Name, Nit: snap.NIT, ContactName: nullable(snap.Contact.Name),
			Email: nullable(snap.Contact.Email), Phone: nullable(snap.Contact.Phone),
			PaymentTermDays: days, Active: snap.Active,
		})
	})
	if err == nil && affected == 0 {
		return domain.ErrSupplierNotFound
	}
	return supplierWriteError("update", err)
}

// termDays converts a payment term to its column; the domain keeps it
// within 0–365.
func termDays(days int) (int32, error) {
	if days < 0 || days > math.MaxInt32 {
		return 0, domain.ErrInvalidPaymentTerm
	}
	return int32(days), nil
}

func supplierWriteError(op string, err error) error {
	switch {
	case db.Violates(err, db.UniqueViolation, supplierNITKey):
		return domain.ErrNITTaken
	case err != nil:
		return fmt.Errorf("purchases: %s supplier: %w", op, err)
	}
	return nil
}

// GetByID returns domain.ErrSupplierNotFound when the supplier does not
// exist in tenantID.
func (r *SupplierRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Supplier, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetSupplierRow, error) {
		return sqlcgen.New(tx).GetSupplier(ctx, id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSupplierNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchases: get supplier: %w", err)
	}
	return supplierFromRow(supplierRow(row))
}

// Search returns one page of the tenant's suppliers matching f, and how many
// match in total, read in the same transaction.
func (r *SupplierRepository) Search(ctx context.Context, f domain.SupplierFilter) ([]*domain.Supplier, int, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	params := sqlcgen.SearchSuppliersParams{
		Active: activeFilter(f.Status), Terms: strings.Fields(f.Text), Nit: nitText(f.Text),
		RowLimit: limit, RowOffset: offset,
	}
	var rows []sqlcgen.SearchSuppliersRow
	var total int64
	err = r.db.WithTenantTx(ctx, f.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.SearchSuppliers(ctx, params); err != nil {
			return err
		}
		total, err = q.CountSuppliers(ctx, sqlcgen.CountSuppliersParams{
			Active: params.Active, Terms: params.Terms, Nit: params.Nit,
		})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("purchases: search suppliers: %w", err)
	}
	out := make([]*domain.Supplier, 0, len(rows))
	for _, row := range rows {
		s, err := supplierFromRow(supplierRow(row))
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, int(total), nil
}

// supplierRow is a stored supplier, as every query reads it.
type supplierRow sqlcgen.GetSupplierRow

func supplierFromRow(row supplierRow) (*domain.Supplier, error) {
	s, err := domain.RehydrateSupplier(domain.SupplierSnapshot{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, NIT: row.Nit,
		Contact:         domain.Contact{Name: value(row.ContactName), Email: value(row.Email), Phone: value(row.Phone)},
		PaymentTermDays: int(row.PaymentTermDays), Active: row.Active, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("purchases: supplier %s: %w", row.ID, err)
	}
	return s, nil
}

// activeFilter is the active column a status asks for; nil for all.
func activeFilter(s domain.SupplierStatus) *bool {
	switch s {
	case domain.SuppliersActive:
		return new(true)
	case domain.SuppliersInactive:
		return new(false)
	default:
		return nil
	}
}

// nitText is text as the start of a stored NIT ("900123456-8"), without the
// dots and spaces people type, or "" if text cannot be one.
func nitText(text string) string {
	nit := strings.NewReplacer(".", "", " ", "").Replace(text)
	if nit == "" || strings.ContainsFunc(nit, func(r rune) bool { return (r < '0' || r > '9') && r != '-' }) {
		return ""
	}
	return nit
}
