package httpadapter

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// ResolutionUseCases are the use cases of the numbering resolutions.
type ResolutionUseCases interface {
	CreateResolution(ctx context.Context, a app.Actor, p domain.ResolutionParams) (app.ResolutionView, error)
	ListResolutions(ctx context.Context, a app.Actor) ([]app.ResolutionView, error)
	DeleteResolution(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// resolutionHandlers are the operations on the numbering resolutions.
type resolutionHandlers struct{ uc ResolutionUseCases }

// ListNumberingResolutions returns the tenant's resolutions.
func (h *resolutionHandlers) ListNumberingResolutions(ctx context.Context, _ invoicingapi.ListNumberingResolutionsRequestObject) (invoicingapi.ListNumberingResolutionsResponseObject, error) {
	views, err := h.uc.ListResolutions(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	out := make(invoicingapi.ListNumberingResolutions200JSONResponse, 0, len(views))
	for _, v := range views {
		out = append(out, resolutionBody(v))
	}
	return out, nil
}

// CreateNumberingResolution registers a resolution.
func (h *resolutionHandlers) CreateNumberingResolution(ctx context.Context, req invoicingapi.CreateNumberingResolutionRequestObject) (invoicingapi.CreateNumberingResolutionResponseObject, error) {
	b := req.Body
	p := domain.ResolutionParams{
		Kind: domain.Kind(b.Kind), Prefix: b.Prefix, Number: b.Number, From: b.RangeFrom, To: b.RangeTo,
		ValidFrom: domain.DateOf(b.ValidFrom.Time), ValidTo: domain.DateOf(b.ValidTo.Time),
	}
	if b.TechnicalKey != nil {
		p.TechnicalKey = *b.TechnicalKey
	}
	v, err := h.uc.CreateResolution(ctx, actor(ctx), p)
	if err != nil {
		return nil, err
	}
	return invoicingapi.CreateNumberingResolution201JSONResponse(resolutionBody(v)), nil
}

// DeleteNumberingResolution removes a resolution that numbered nothing.
func (h *resolutionHandlers) DeleteNumberingResolution(ctx context.Context, req invoicingapi.DeleteNumberingResolutionRequestObject) (invoicingapi.DeleteNumberingResolutionResponseObject, error) {
	if err := h.uc.DeleteResolution(ctx, actor(ctx), req.ResolutionId); err != nil {
		return nil, err
	}
	return invoicingapi.DeleteNumberingResolution204Response{}, nil
}

func resolutionBody(v app.ResolutionView) invoicingapi.NumberingResolution {
	r := v.Resolution
	p := r.Params()
	return invoicingapi.NumberingResolution{
		Id: r.ID(), Kind: invoicingapi.ElectronicDocumentKind(p.Kind), Prefix: p.Prefix, Number: p.Number,
		TechnicalKey: p.TechnicalKey, RangeFrom: p.From, RangeTo: p.To,
		ValidFrom: openapi_types.Date{Time: p.ValidFrom.Time()}, ValidTo: openapi_types.Date{Time: p.ValidTo.Time()},
		NextNumber: r.Next(), Remaining: r.Remaining(), NearEnd: v.NearEnd,
	}
}
