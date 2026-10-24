package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestUnits(t *testing.T) {
	t.Run("lista las unidades", func(t *testing.T) {
		uc := &stubUseCases{units: []*domain.UnitOfMeasure{meters, mustUnit("und", "Unidad")}}

		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/units", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items":[
			{"id":"`+meters.ID().String()+`","code":"m","name":"Metro"},
			{"id":"`+uc.units[1].ID().String()+`","code":"und","name":"Unidad"}]}`, rec.Body.String())
	})

	t.Run("crea una unidad con los datos y el actor del token", func(t *testing.T) {
		uc := &stubUseCases{unit: mustUnit("par", "Par")}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/units", `{"code":"PAR","name":"Par"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID}, uc.actor)
		require.Equal(t, "PAR", uc.code)
		require.Equal(t, "Par", uc.name)
		require.Equal(t, "par", decodeMap(t, rec)["code"])
	})

	t.Run("edita una unidad", func(t *testing.T) {
		uc := &stubUseCases{unit: mustUnit("pares", "Pares")}
		id := uuid.Must(uuid.NewV7())

		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/units/"+id.String(), `{"code":"pares","name":"Pares"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
		require.Equal(t, "pares", decodeMap(t, rec)["code"])
	})

	t.Run("borra una unidad", func(t *testing.T) {
		uc := &stubUseCases{}
		id := uuid.Must(uuid.NewV7())

		rec := serve(t, uc, owner, http.MethodDelete, "/api/v1/units/"+id.String(), "")

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
	})

	t.Run("mapea los errores de las unidades", func(t *testing.T) {
		tests := []struct {
			method, path, body string
			err                error
			status             int
			code               string
		}{
			{http.MethodPost, "/api/v1/units", `{"code":"m","name":"Metro"}`, domain.ErrUnitCodeTaken, http.StatusConflict, "unit_code_taken"},
			{http.MethodPost, "/api/v1/units", `{"code":"caja x12","name":"Caja"}`, domain.ErrInvalidUnitCode, http.StatusUnprocessableEntity, "invalid_unit_code"},
			{http.MethodPost, "/api/v1/units", `{"code":"m","name":" "}`, domain.ErrInvalidUnitName, http.StatusUnprocessableEntity, "invalid_unit_name"},
			{http.MethodPut, "/api/v1/units/" + uuid.Must(uuid.NewV7()).String(), `{"code":"m","name":"Metro"}`, domain.ErrUnitNotFound, http.StatusNotFound, "unit_not_found"},
			{http.MethodDelete, "/api/v1/units/" + uuid.Must(uuid.NewV7()).String(), "", domain.ErrUnitInUse, http.StatusConflict, "unit_in_use"},
			{http.MethodDelete, "/api/v1/units/" + uuid.Must(uuid.NewV7()).String(), "", domain.ErrUnitNotFound, http.StatusNotFound, "unit_not_found"},
		}
		for _, tt := range tests {
			t.Run(tt.method+" "+tt.code, func(t *testing.T) {
				rec := serve(t, &stubUseCases{err: tt.err}, owner, tt.method, tt.path, tt.body)
				requireError(t, rec, tt.status, tt.code)
			})
		}
	})
}
