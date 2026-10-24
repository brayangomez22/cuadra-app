package httpadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// VariantAxisUseCases are the use cases of the variant axes.
type VariantAxisUseCases interface {
	CreateVariantAxis(ctx context.Context, a app.Actor, name string, values []string) (*domain.VariantAxis, error)
	UpdateVariantAxis(ctx context.Context, a app.Actor, id uuid.UUID, values []string) (*domain.VariantAxis, error)
	ListVariantAxes(ctx context.Context, a app.Actor) ([]*domain.VariantAxis, error)
	DeleteVariantAxis(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// variantAxisHandlers are the operations on variant axes.
type variantAxisHandlers struct{ uc VariantAxisUseCases }

// ListVariantAxes lists the tenant's variant axes.
func (h *variantAxisHandlers) ListVariantAxes(ctx context.Context, _ catalogapi.ListVariantAxesRequestObject) (catalogapi.ListVariantAxesResponseObject, error) {
	axes, err := h.uc.ListVariantAxes(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]catalogapi.VariantAxis, 0, len(axes))
	for _, a := range axes {
		items = append(items, axisBody(a))
	}
	return catalogapi.ListVariantAxes200JSONResponse{Items: items}, nil
}

// CreateVariantAxis defines a variant axis.
func (h *variantAxisHandlers) CreateVariantAxis(ctx context.Context, req catalogapi.CreateVariantAxisRequestObject) (catalogapi.CreateVariantAxisResponseObject, error) {
	a, err := h.uc.CreateVariantAxis(ctx, actor(ctx), req.Body.Name, req.Body.Values)
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateVariantAxis201JSONResponse(axisBody(a)), nil
}

// UpdateVariantAxis replaces the values an axis allows.
func (h *variantAxisHandlers) UpdateVariantAxis(ctx context.Context, req catalogapi.UpdateVariantAxisRequestObject) (catalogapi.UpdateVariantAxisResponseObject, error) {
	a, err := h.uc.UpdateVariantAxis(ctx, actor(ctx), req.AxisId, req.Body.Values)
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateVariantAxis200JSONResponse(axisBody(a)), nil
}

// DeleteVariantAxis removes an axis no variant uses.
func (h *variantAxisHandlers) DeleteVariantAxis(ctx context.Context, req catalogapi.DeleteVariantAxisRequestObject) (catalogapi.DeleteVariantAxisResponseObject, error) {
	if err := h.uc.DeleteVariantAxis(ctx, actor(ctx), req.AxisId); err != nil {
		return nil, err
	}
	return catalogapi.DeleteVariantAxis204Response{}, nil
}

func axisBody(a *domain.VariantAxis) catalogapi.VariantAxis {
	return catalogapi.VariantAxis{Id: a.ID(), Name: a.Name(), Values: a.Values()}
}

// describeAxis names the axis of an *AxisError in the message and in
// details.axis.
func describeAxis(format func(axis string) string) func(error) (string, map[string]string, bool) {
	return func(err error) (string, map[string]string, bool) {
		var axisErr *domain.AxisError
		if !errors.As(err, &axisErr) {
			return "", nil, false
		}
		return format(axisErr.Axis), map[string]string{"axis": axisErr.Axis}, true
	}
}
