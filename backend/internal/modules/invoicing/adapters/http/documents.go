package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// DocumentUseCases are the use cases of the electronic documents.
type DocumentUseCases interface {
	ListDocuments(ctx context.Context, a app.Actor, f domain.DocumentFilter) (domain.DocumentPage, error)
	GetDocument(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Document, error)
	RetryDocument(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Document, error)
}

// documentHandlers are the operations on the electronic documents.
type documentHandlers struct{ uc DocumentUseCases }

// ListElectronicDocuments returns a page of the tenant's documents.
func (h *documentHandlers) ListElectronicDocuments(ctx context.Context, req invoicingapi.ListElectronicDocumentsRequestObject) (invoicingapi.ListElectronicDocumentsResponseObject, error) {
	f := domain.DocumentFilter{SaleID: req.Params.SaleId}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	if req.Params.Status != nil {
		s := domain.Status(*req.Params.Status)
		f.Status = &s
	}
	p, err := h.uc.ListDocuments(ctx, actor(ctx), f)
	if err != nil {
		return nil, err
	}
	items := make([]invoicingapi.ElectronicDocument, 0, len(p.Documents))
	for _, d := range p.Documents {
		items = append(items, documentBody(d))
	}
	return invoicingapi.ListElectronicDocuments200JSONResponse{Items: items, Total: p.Total}, nil
}

// GetElectronicDocument returns a document.
func (h *documentHandlers) GetElectronicDocument(ctx context.Context, req invoicingapi.GetElectronicDocumentRequestObject) (invoicingapi.GetElectronicDocumentResponseObject, error) {
	d, err := h.uc.GetDocument(ctx, actor(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return invoicingapi.GetElectronicDocument200JSONResponse(documentBody(d)), nil
}

// RetryElectronicDocument sends a pending or rejected document again.
func (h *documentHandlers) RetryElectronicDocument(ctx context.Context, req invoicingapi.RetryElectronicDocumentRequestObject) (invoicingapi.RetryElectronicDocumentResponseObject, error) {
	d, err := h.uc.RetryDocument(ctx, actor(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return invoicingapi.RetryElectronicDocument200JSONResponse(documentBody(d)), nil
}

func documentBody(d *domain.Document) invoicingapi.ElectronicDocument {
	reasons := d.Reasons()
	if reasons == nil {
		reasons = []string{}
	}
	return invoicingapi.ElectronicDocument{
		Id: d.ID(), SaleId: d.SaleID(), Kind: invoicingapi.ElectronicDocumentKind(d.Kind()),
		Prefix: d.Prefix(), Number: d.Number(), Status: invoicingapi.ElectronicDocumentStatus(d.Status()),
		IssuedAt: d.IssuedAt(), Attempts: d.Attempts(), ProviderId: optional(d.ProviderID()),
		Code: optional(d.Code()), Qr: optional(d.QR()), Reasons: reasons, LastError: optional(d.LastError()),
		AcceptedAt: d.AcceptedAt(), UpdatedAt: d.UpdatedAt(),
	}
}
