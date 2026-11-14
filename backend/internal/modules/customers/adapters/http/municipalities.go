package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http/customersapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// MunicipalityUseCases are the use cases of the DANE's municipalities.
type MunicipalityUseCases interface {
	ListMunicipalities(ctx context.Context, a app.Actor) ([]domain.Municipality, error)
}

type municipalityHandlers struct{ uc MunicipalityUseCases }

// ListMunicipalities lists the DANE's municipalities.
func (h *municipalityHandlers) ListMunicipalities(ctx context.Context, _ customersapi.ListMunicipalitiesRequestObject) (customersapi.ListMunicipalitiesResponseObject, error) {
	list, err := h.uc.ListMunicipalities(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]customersapi.Municipality, 0, len(list))
	for _, m := range list {
		items = append(items, customersapi.Municipality{
			Code: m.Code, Name: m.Name, DepartmentCode: m.DepartmentCode, DepartmentName: m.DepartmentName,
		})
	}
	return customersapi.ListMunicipalities200JSONResponse{Items: items}, nil
}
