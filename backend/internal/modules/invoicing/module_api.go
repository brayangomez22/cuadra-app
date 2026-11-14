package invoicing

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
)

// SaleDocumentRequest is a sale being confirmed (see app.SaleDocumentRequest).
type SaleDocumentRequest = app.SaleDocumentRequest

// RegisterSale gives a sale userID confirms its electronic document (see
// app.Service.RegisterSale). It is for the sales module, through the port
// internal/server adapts; with the sale's ctx it joins its transaction.
func (m *Module) RegisterSale(ctx context.Context, tenantID, userID uuid.UUID, in SaleDocumentRequest) error {
	return m.service.RegisterSale(ctx, app.Actor{TenantID: tenantID, UserID: userID}, in)
}

// HasDocument is whether the sale has an electronic document, for the sales
// module's void.
func (m *Module) HasDocument(ctx context.Context, tenantID, userID, saleID uuid.UUID) (bool, error) {
	return m.service.HasDocument(ctx, app.Actor{TenantID: tenantID, UserID: userID}, saleID)
}
