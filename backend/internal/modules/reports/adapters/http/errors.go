package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the reports' domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrInvalidPeriod, Status: http.StatusUnprocessableEntity, Code: "invalid_period",
		Message: "El periodo no es válido: el último día debe ser igual o posterior al primero y el periodo no puede pasar de un año."},
	{Err: domain.ErrInvalidFilter, Status: http.StatusUnprocessableEntity, Code: "invalid_filter",
		Message: "Revise los filtros del reporte."},
	{Err: domain.ErrCostsHidden, Status: http.StatusForbidden, Code: "costs_hidden",
		Message: "Este reporte muestra costos y utilidad, y su rol no los ve."},
}
