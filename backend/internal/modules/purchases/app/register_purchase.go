package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// PurchaseLineInput is a line of the supplier's invoice. UnitID is an
// alternate unit of the variant, or nil for the base unit; TaxRate defaults
// to the product's.
type PurchaseLineInput struct {
	VariantID uuid.UUID
	UnitID    *uuid.UUID
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
	TaxRate   *decimal.Decimal
}

// PurchaseInput is a supplier's invoice to register.
type PurchaseInput struct {
	SupplierID    uuid.UUID
	LocationID    uuid.UUID
	InvoiceNumber string
	IssuedOn      domain.Date
	DueOn         *domain.Date
	Payment       domain.Payment
	Notes         string
	Lines         []PurchaseLineInput
}

// LineError is the error of one line of a purchase (its index, from 0); it
// unwraps to the domain error, so the client can mark the line.
type LineError struct {
	Index int
	Err   error
}

func (e *LineError) Error() string { return fmt.Sprintf("line %d: %v", e.Index, e.Err) }

func (e *LineError) Unwrap() error { return e.Err }

// RegisterPurchase registers a supplier's invoice: its lines, in the unit
// each was bought in, enter the stock of the location (updating the average
// cost) in the purchase's transaction, so a failed entry leaves no purchase.
// The supplier's invoice is registered once (ErrInvoiceTaken).
func (s *Service) RegisterPurchase(ctx context.Context, a Actor, in PurchaseInput) (_ *domain.Purchase, err error) {
	ctx, span := s.startSpan(ctx, "RegisterPurchase", a, supplierAttr(in.SupplierID),
		attribute.String("location.id", in.LocationID.String()), attribute.Int("purchase.lines", len(in.Lines)))
	defer func() { usecase.End(span, err) }()

	supplier, err := s.Suppliers.GetByID(ctx, a.TenantID, in.SupplierID)
	if err != nil {
		return nil, err
	}
	lines, err := s.purchaseLines(ctx, a, in.Lines)
	if err != nil {
		return nil, err
	}
	p, err := domain.NewPurchase(a.TenantID, domain.PurchaseParams{
		Supplier: supplier, LocationID: in.LocationID, InvoiceNumber: in.InvoiceNumber, IssuedOn: in.IssuedOn,
		DueOn: in.DueOn, Payment: in.Payment, Notes: in.Notes, CreatedBy: a.UserID,
	}, lines, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(purchaseAttr(p.ID()))
	err = s.Purchases.Save(ctx, p, func(ctx context.Context) error {
		received, err := s.Stock.ReceivePurchase(ctx, a.TenantID, a.UserID, p.LocationID(), p.ID(), stockLines(p))
		if err != nil {
			return err
		}
		return p.Receive(received)
	})
	if errors.Is(err, domain.ErrInvoiceTaken) {
		s.duplicateInvoices.Add(ctx, 1)
	}
	if err != nil {
		return nil, err
	}
	s.recordPurchase(ctx, p)
	return p, nil
}

// purchaseLines completes each line with what the catalog says of its
// variant: the names, the unit's code and factor and the default VAT rate.
// An error names the line.
func (s *Service) purchaseLines(ctx context.Context, a Actor, in []PurchaseLineInput) ([]domain.LineInput, error) {
	ids := make([]uuid.UUID, 0, len(in))
	for _, l := range in {
		if !slices.Contains(ids, l.VariantID) {
			ids = append(ids, l.VariantID)
		}
	}
	found, err := s.Catalog.Variants(ctx, a.TenantID, ids)
	if err != nil {
		return nil, err
	}
	variants := make(map[uuid.UUID]VariantInfo, len(found))
	for _, v := range found {
		variants[v.ID] = v
	}
	out := make([]domain.LineInput, len(in))
	for i, l := range in {
		if out[i], err = lineInput(l, variants); err != nil {
			return nil, &LineError{Index: i, Err: err}
		}
	}
	return out, nil
}

func lineInput(l PurchaseLineInput, variants map[uuid.UUID]VariantInfo) (domain.LineInput, error) {
	v, ok := variants[l.VariantID]
	if !ok {
		return domain.LineInput{}, domain.ErrVariantNotFound
	}
	if !v.Active {
		return domain.LineInput{}, domain.ErrVariantInactive
	}
	unit := domain.LineUnit{Code: v.BaseUnitCode, Factor: decimal.NewFromInt(1)}
	if l.UnitID != nil {
		i := slices.IndexFunc(v.Units, func(u UnitInfo) bool { return u.UnitID == *l.UnitID })
		if i < 0 {
			return domain.LineInput{}, domain.ErrUnitNotAvailable
		}
		id := v.Units[i].UnitID
		unit = domain.LineUnit{ID: &id, Code: v.Units[i].Code, Factor: v.Units[i].Factor}
	}
	rate := v.TaxRate
	if l.TaxRate != nil {
		rate = *l.TaxRate
	}
	return domain.LineInput{
		VariantID: v.ID, ProductName: v.ProductName, VariantLabel: v.Label, Unit: unit,
		Quantity: l.Quantity, UnitCost: l.UnitCost, TaxRate: rate,
	}, nil
}

// stockLines is what each line of p puts into the stock, in line order.
func stockLines(p *domain.Purchase) []StockLine {
	lines := p.Lines()
	out := make([]StockLine, len(lines))
	for i, l := range lines {
		out[i] = StockLine{VariantID: l.VariantID(), Quantity: l.Quantity(), UnitCost: l.UnitCost(), Factor: l.Unit().Factor}
	}
	return out
}

// recordPurchase counts a registered purchase and its lines by unit.
func (s *Service) recordPurchase(ctx context.Context, p *domain.Purchase) {
	s.registered.Add(ctx, 1, metric.WithAttributes(attribute.String("payment", string(p.Payment()))))
	for _, l := range p.Lines() {
		unit := "base"
		if l.Unit().ID != nil {
			unit = "alternate"
		}
		s.lines.Add(ctx, 1, metric.WithAttributes(attribute.String("unit", unit)))
	}
}
