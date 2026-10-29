package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// PurchaseRepository implements domain.PurchaseRepository.
type PurchaseRepository struct {
	db *db.DB
}

// NewPurchaseRepository returns a PurchaseRepository on d.
func NewPurchaseRepository(d *db.DB) *PurchaseRepository { return &PurchaseRepository{db: d} }

// Save stores a new purchase in one transaction, in this order: the header
// (its unique invoice is checked here, before touching the stock), then
// receive (which joins the transaction through ctx and records on p what
// each line put into the stock), then the lines with what was received.
func (r *PurchaseRepository) Save(ctx context.Context, p *domain.Purchase, receive func(context.Context) error) error {
	// receiveErr keeps the inventory's error apart from the database errors,
	// which are translated and wrapped.
	var receiveErr error
	err := r.db.InTenantTx(ctx, p.TenantID(), func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.InsertPurchase(ctx, purchaseParams(p)); err != nil {
			return err
		}
		if err := receive(ctx); err != nil {
			receiveErr = err
			return err
		}
		return insertLines(ctx, q, p)
	})
	switch {
	case receiveErr != nil:
		return receiveErr
	case db.Violates(err, db.UniqueViolation, purchaseInvoiceKey):
		return domain.ErrInvoiceTaken
	case db.Violates(err, db.ForeignKeyViolation, purchaseSupplierFKey):
		return domain.ErrSupplierNotFound
	case db.Violates(err, db.ForeignKeyViolation, purchaseLocationFKey):
		return domain.ErrLocationNotFound
	case db.Violates(err, db.ForeignKeyViolation, purchaseVariantFKey):
		return domain.ErrVariantNotFound
	case db.Violates(err, db.ForeignKeyViolation, purchaseUnitFKey):
		return domain.ErrUnitNotAvailable
	case err != nil:
		return fmt.Errorf("purchases: save purchase %s: %w", p.ID(), err)
	}
	return nil
}

func purchaseParams(p *domain.Purchase) sqlcgen.InsertPurchaseParams {
	t := p.Totals()
	params := sqlcgen.InsertPurchaseParams{
		ID: p.ID(), TenantID: p.TenantID(), SupplierID: p.SupplierID(), LocationID: p.LocationID(),
		InvoiceNumber: p.InvoiceNumber(), InvoiceKey: p.InvoiceKey(), IssuedOn: p.IssuedOn().Time(),
		Payment: string(p.Payment()), Notes: nullable(p.Notes()), Subtotal: t.Subtotal, Tax: t.Tax, Total: t.Total,
		BalanceDue: p.BalanceDue(), CreatedBy: p.CreatedBy(), CreatedAt: p.CreatedAt(),
	}
	if due := p.DueOn(); due != nil {
		params.DueOn = new(due.Time())
	}
	return params
}

func insertLines(ctx context.Context, q *sqlcgen.Queries, p *domain.Purchase) error {
	for i, l := range p.Lines() {
		u := l.Unit()
		err := q.InsertPurchaseLine(ctx, sqlcgen.InsertPurchaseLineParams{
			TenantID: p.TenantID(), PurchaseID: p.ID(), Position: int32(i),
			VariantID: l.VariantID(), ProductName: l.ProductName(), VariantLabel: nullable(l.VariantLabel()),
			UnitID: u.ID, UnitCode: nullable(u.Code), UnitFactor: u.Factor, Quantity: l.Quantity(),
			UnitCost: l.UnitCost(), TaxRate: l.TaxRate(), Subtotal: l.Subtotal(), Tax: l.Tax(),
			BaseQuantity: l.BaseQuantity(), BaseUnitCost: l.BaseUnitCost(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetByID returns domain.ErrPurchaseNotFound when the purchase does not
// exist in tenantID.
func (r *PurchaseRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Purchase, error) {
	var row sqlcgen.GetPurchaseRow
	var lines []sqlcgen.PurchaseLinesRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if row, err = q.GetPurchase(ctx, id); err != nil {
			return err
		}
		lines, err = q.PurchaseLines(ctx, id)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchases: get purchase: %w", err)
	}
	return purchaseFromRows(row, lines)
}

func purchaseFromRows(row sqlcgen.GetPurchaseRow, lines []sqlcgen.PurchaseLinesRow) (*domain.Purchase, error) {
	s := domain.PurchaseSnapshot{
		ID: row.ID, TenantID: row.TenantID, SupplierID: row.SupplierID, LocationID: row.LocationID,
		InvoiceNumber: row.InvoiceNumber, IssuedOn: domain.DateOf(row.IssuedOn), Payment: domain.Payment(row.Payment),
		Notes: value(row.Notes), BalanceDue: row.BalanceDue, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
	}
	if row.DueOn != nil {
		s.DueOn = new(domain.DateOf(*row.DueOn))
	}
	for _, l := range lines {
		s.Lines = append(s.Lines, domain.LineSnapshot{
			LineInput: domain.LineInput{
				VariantID: l.VariantID, ProductName: l.ProductName, VariantLabel: value(l.VariantLabel),
				Unit:     domain.LineUnit{ID: l.UnitID, Code: value(l.UnitCode), Factor: l.UnitFactor},
				Quantity: l.Quantity, UnitCost: l.UnitCost, TaxRate: l.TaxRate,
			},
			Received: domain.Received{BaseQuantity: l.BaseQuantity, BaseUnitCost: l.BaseUnitCost},
		})
	}
	p, err := domain.RehydratePurchase(s)
	if err != nil {
		return nil, fmt.Errorf("purchases: purchase %s: %w", row.ID, err)
	}
	return p, nil
}
