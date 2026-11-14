package httpadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// UnitUseCases are the use cases of the units of measure.
type UnitUseCases interface {
	ListUnits(ctx context.Context, a app.Actor) ([]*domain.UnitOfMeasure, error)
	CreateUnit(ctx context.Context, a app.Actor, in app.UnitInput) (*domain.UnitOfMeasure, error)
	UpdateUnit(ctx context.Context, a app.Actor, id uuid.UUID, in app.UnitInput) (*domain.UnitOfMeasure, error)
	DeleteUnit(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// unitHandlers are the operations on units of measure.
type unitHandlers struct{ uc UnitUseCases }

// errUnitPathNotFound is domain.ErrUnitNotFound when the unit is the
// resource of the path (404). As the unit of a product it is a field of the
// body, and stays a 422.
var errUnitPathNotFound = errors.New("catalog: unit of the path not found")

// unitPathError turns a missing unit into errUnitPathNotFound.
func unitPathError(err error) error {
	if errors.Is(err, domain.ErrUnitNotFound) {
		return fmt.Errorf("%w: %w", errUnitPathNotFound, err)
	}
	return err
}

// ListUnits lists the tenant's units of measure.
func (h *unitHandlers) ListUnits(ctx context.Context, _ catalogapi.ListUnitsRequestObject) (catalogapi.ListUnitsResponseObject, error) {
	units, err := h.uc.ListUnits(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]catalogapi.UnitOfMeasure, 0, len(units))
	for _, u := range units {
		items = append(items, unitBody(u.Ref()))
	}
	return catalogapi.ListUnits200JSONResponse{Items: items}, nil
}

// CreateUnit adds a unit of measure.
func (h *unitHandlers) CreateUnit(ctx context.Context, req catalogapi.CreateUnitRequestObject) (catalogapi.CreateUnitResponseObject, error) {
	u, err := h.uc.CreateUnit(ctx, actor(ctx), unitInput(*req.Body))
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateUnit201JSONResponse(unitBody(u.Ref())), nil
}

// UpdateUnit changes a unit's code, name and DIAN code.
func (h *unitHandlers) UpdateUnit(ctx context.Context, req catalogapi.UpdateUnitRequestObject) (catalogapi.UpdateUnitResponseObject, error) {
	u, err := h.uc.UpdateUnit(ctx, actor(ctx), req.UnitId, unitInput(*req.Body))
	if err != nil {
		return nil, unitPathError(err)
	}
	return catalogapi.UpdateUnit200JSONResponse(unitBody(u.Ref())), nil
}

// DeleteUnit removes a unit no product uses.
func (h *unitHandlers) DeleteUnit(ctx context.Context, req catalogapi.DeleteUnitRequestObject) (catalogapi.DeleteUnitResponseObject, error) {
	if err := h.uc.DeleteUnit(ctx, actor(ctx), req.UnitId); err != nil {
		return nil, unitPathError(err)
	}
	return catalogapi.DeleteUnit204Response{}, nil
}

func unitBody(u domain.UnitRef) catalogapi.UnitOfMeasure {
	body := catalogapi.UnitOfMeasure{Id: u.ID, Code: u.Code, Name: u.Name}
	if u.DIANCode != "" {
		body.DianCode = &u.DIANCode
	}
	return body
}

func unitInput(b catalogapi.UnitInput) app.UnitInput {
	in := app.UnitInput{Code: b.Code, Name: b.Name}
	if b.DianCode != nil {
		in.DIANCode = *b.DianCode
	}
	return in
}
