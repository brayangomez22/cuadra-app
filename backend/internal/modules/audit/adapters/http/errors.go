package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the audit domain errors to HTTP. An unknown action does
// not reach the use case (the spec's enum rejects it first).
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrInvalidFilter, Status: http.StatusUnprocessableEntity, Code: "invalid_filter",
		Message: "La fecha inicial debe ser anterior o igual a la final."},
	{Err: domain.ErrUnknownAction, Status: http.StatusBadRequest, Code: "invalid_request",
		Message: "La acción del filtro no es válida."},
}
