package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http/purchasesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// PurchaseUseCases are the use cases of the purchases.
type PurchaseUseCases interface {
	RegisterPurchase(ctx context.Context, a app.Actor, in app.PurchaseInput) (*domain.Purchase, error)
	GetPurchase(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Purchase, error)
	ListPurchases(ctx context.Context, a app.Actor, q app.PurchaseQuery) (app.PurchasePage, error)
}

// purchaseHandlers are the operations on purchases.
type purchaseHandlers struct{ uc PurchaseUseCases }

// ListPurchases lists the tenant's purchases.
func (h *purchaseHandlers) ListPurchases(ctx context.Context, req purchasesapi.ListPurchasesRequestObject) (purchasesapi.ListPurchasesResponseObject, error) {
	p := req.Params
	q := app.PurchaseQuery{SupplierID: p.SupplierId, From: dateOf(p.From), To: dateOf(p.To), Limit: defaultPageSize}
	if p.Status != nil && *p.Status != purchasesapi.ListPurchasesParamsStatusAll {
		q.Status = domain.PurchaseStatus(*p.Status)
	}
	if p.Limit != nil {
		q.Limit = *p.Limit
	}
	if p.Offset != nil {
		q.Offset = *p.Offset
	}
	page, err := h.uc.ListPurchases(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	items := make([]purchasesapi.PurchaseSummary, 0, len(page.Items))
	for _, s := range page.Items {
		items = append(items, summaryBody(s))
	}
	return purchasesapi.ListPurchases200JSONResponse{Items: items, Total: page.Total}, nil
}

// RegisterPurchase registers a supplier's invoice.
func (h *purchaseHandlers) RegisterPurchase(ctx context.Context, req purchasesapi.RegisterPurchaseRequestObject) (purchasesapi.RegisterPurchaseResponseObject, error) {
	in, err := purchaseInput(*req.Body)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.RegisterPurchase(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return purchasesapi.RegisterPurchase201JSONResponse(purchaseBody(p)), nil
}

// GetPurchase returns a purchase with its lines.
func (h *purchaseHandlers) GetPurchase(ctx context.Context, req purchasesapi.GetPurchaseRequestObject) (purchasesapi.GetPurchaseResponseObject, error) {
	p, err := h.uc.GetPurchase(ctx, actor(ctx), req.PurchaseId)
	if err != nil {
		return nil, err
	}
	return purchasesapi.GetPurchase200JSONResponse(purchaseBody(p)), nil
}

// purchaseInput translates the body. An amount that cannot be read is the
// domain's error of its line.
func purchaseInput(b purchasesapi.PurchaseInput) (app.PurchaseInput, error) {
	in := app.PurchaseInput{
		SupplierID: b.SupplierId, LocationID: b.LocationId, InvoiceNumber: b.InvoiceNumber,
		IssuedOn: domain.DateOf(b.IssuedOn.Time), DueOn: dateOf(b.DueOn), Payment: domain.Payment(b.Payment),
		Notes: text(b.Notes), Lines: make([]app.PurchaseLineInput, len(b.Lines)),
	}
	for i, l := range b.Lines {
		line, err := lineInput(l)
		if err != nil {
			return app.PurchaseInput{}, &app.LineError{Index: i, Err: err}
		}
		in.Lines[i] = line
	}
	return in, nil
}

func lineInput(l purchasesapi.PurchaseLineInput) (app.PurchaseLineInput, error) {
	qty, err := httpx.ParseDecimal(l.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return app.PurchaseLineInput{}, err
	}
	cost, err := httpx.ParseDecimal(l.UnitCost, domain.ErrInvalidUnitCost)
	if err != nil {
		return app.PurchaseLineInput{}, err
	}
	out := app.PurchaseLineInput{VariantID: l.VariantId, UnitID: l.UnitId, Quantity: qty, UnitCost: cost}
	if l.TaxRate != nil {
		rate, err := httpx.ParseDecimal(*l.TaxRate, domain.ErrInvalidTaxRate)
		if err != nil {
			return app.PurchaseLineInput{}, err
		}
		out.TaxRate = &rate
	}
	return out, nil
}
