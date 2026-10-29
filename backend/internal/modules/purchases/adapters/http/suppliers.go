package httpadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http/purchasesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// SupplierUseCases are the use cases of the suppliers.
type SupplierUseCases interface {
	CreateSupplier(ctx context.Context, a app.Actor, p domain.SupplierParams) (app.SupplierView, error)
	UpdateSupplier(ctx context.Context, a app.Actor, id uuid.UUID, p domain.SupplierParams) (app.SupplierView, error)
	GetSupplier(ctx context.Context, a app.Actor, id uuid.UUID) (app.SupplierView, error)
	ListSuppliers(ctx context.Context, a app.Actor, q app.SupplierQuery) (app.SupplierPage, error)
}

// supplierHandlers are the operations on suppliers.
type supplierHandlers struct{ uc SupplierUseCases }

// ListSuppliers searches the tenant's suppliers.
func (h *supplierHandlers) ListSuppliers(ctx context.Context, req purchasesapi.ListSuppliersRequestObject) (purchasesapi.ListSuppliersResponseObject, error) {
	q := app.SupplierQuery{Text: text(req.Params.Q), Status: domain.SuppliersActive, Limit: defaultPageSize}
	if req.Params.Status != nil {
		q.Status = domain.SupplierStatus(*req.Params.Status)
	}
	if req.Params.Limit != nil {
		q.Limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		q.Offset = *req.Params.Offset
	}
	page, err := h.uc.ListSuppliers(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	items := make([]purchasesapi.Supplier, 0, len(page.Items))
	for _, s := range page.Items {
		items = append(items, supplierBody(s))
	}
	return purchasesapi.ListSuppliers200JSONResponse{Items: items, Total: page.Total}, nil
}

// CreateSupplier adds a supplier.
func (h *supplierHandlers) CreateSupplier(ctx context.Context, req purchasesapi.CreateSupplierRequestObject) (purchasesapi.CreateSupplierResponseObject, error) {
	s, err := h.uc.CreateSupplier(ctx, actor(ctx), supplierParams(*req.Body))
	if err != nil {
		return nil, err
	}
	return purchasesapi.CreateSupplier201JSONResponse(supplierBody(s)), nil
}

// GetSupplier returns a supplier and what it is owed.
func (h *supplierHandlers) GetSupplier(ctx context.Context, req purchasesapi.GetSupplierRequestObject) (purchasesapi.GetSupplierResponseObject, error) {
	s, err := h.uc.GetSupplier(ctx, actor(ctx), req.SupplierId)
	if err != nil {
		return nil, supplierPathError(err)
	}
	return purchasesapi.GetSupplier200JSONResponse(supplierBody(s)), nil
}

// UpdateSupplier replaces a supplier's data.
func (h *supplierHandlers) UpdateSupplier(ctx context.Context, req purchasesapi.UpdateSupplierRequestObject) (purchasesapi.UpdateSupplierResponseObject, error) {
	s, err := h.uc.UpdateSupplier(ctx, actor(ctx), req.SupplierId, supplierParams(*req.Body))
	if err != nil {
		return nil, supplierPathError(err)
	}
	return purchasesapi.UpdateSupplier200JSONResponse(supplierBody(s)), nil
}

// supplierPathError turns the missing supplier of the path into a 404.
func supplierPathError(err error) error {
	if errors.Is(err, domain.ErrSupplierNotFound) {
		return errSupplierPathNotFound
	}
	return err
}

// supplierParams translates the body; active defaults to true and the term
// to 0, as in the contract.
func supplierParams(b purchasesapi.SupplierInput) domain.SupplierParams {
	p := domain.SupplierParams{
		Name: b.Name, NIT: b.Nit, Active: true,
		Contact: domain.Contact{Name: text(b.ContactName), Email: text(b.Email), Phone: text(b.Phone)},
	}
	if b.Active != nil {
		p.Active = *b.Active
	}
	if b.PaymentTermDays != nil {
		p.PaymentTermDays = *b.PaymentTermDays
	}
	return p
}

func supplierBody(s app.SupplierView) purchasesapi.Supplier {
	c := s.Contact()
	return purchasesapi.Supplier{
		Id: s.ID(), Name: s.Name(), Nit: s.NIT().String(), ContactName: optional(c.Name), Email: optional(c.Email),
		Phone: optional(c.Phone), PaymentTermDays: s.PaymentTermDays(), Active: s.Active(),
		BalanceDue: httpx.FormatMoney(s.BalanceDue), CreatedAt: s.CreatedAt(),
	}
}
