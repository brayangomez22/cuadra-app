package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

var withVariantsBody = `{
	"name": "Camiseta básica", "base_unit_id": "` + meters.ID().String() + `", "tax_rate": "0.19", "attributes": {},
	"variants": [
		{"sku": "cam-s", "cost": "15000", "price": "30000", "axis_values": {"talla": "S"}},
		{"sku": "cam-m", "barcode": "7700000000002", "cost": "15000", "price": "32000", "axis_values": {"talla": "M"}}
	]
}`

func TestCreateProductWithVariants(t *testing.T) {
	t.Run("pasa el producto y sus variantes al caso de uso y responde 201", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "15000", "30000")}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products-with-variants", withVariantsBody)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, "Camiseta básica", uc.withVars.Name)
		require.Len(t, uc.withVars.Variants, 2)
		second := uc.withVars.Variants[1]
		require.Equal(t, "cam-m", second.SKU)
		require.Equal(t, "7700000000002", second.Barcode)
		require.Equal(t, map[string]string{"talla": "M"}, second.AxisValues)
		require.True(t, decimal.RequireFromString("32000").Equal(second.Price))
	})

	t.Run("variantes repetidas responden 422 duplicate_variant", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrDuplicateVariant}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products-with-variants", withVariantsBody)

		requireError(t, rec, http.StatusUnprocessableEntity, "duplicate_variant")
	})

	t.Run("un SKU o un código de barras tomado responde 409 con el código en details", func(t *testing.T) {
		tests := []struct {
			err         error
			code, field string
			value       string
		}{
			{domain.ErrSKUTaken, "sku_taken", "sku", "CAM-M"},
			{domain.ErrBarcodeTaken, "barcode_taken", "barcode", "7700000000002"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				uc := &stubUseCases{err: &domain.CodeTakenError{Code: tt.value, Err: tt.err}}

				rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products-with-variants", withVariantsBody)

				requireError(t, rec, http.StatusConflict, tt.code)
				body := decodeMap(t, rec)["error"].(map[string]any)
				require.Equal(t, map[string]any{tt.field: tt.value}, body["details"])
				require.Contains(t, body["message"], "«"+tt.value+"»")
			})
		}
	})
}

func TestUpdateProductBaseUnitLocked(t *testing.T) {
	t.Run("cambiar la unidad base de un producto con movimientos responde 409 base_unit_locked", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrBaseUnitLocked}

		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/products/"+uuid.Must(uuid.NewV7()).String(), productBody)

		requireError(t, rec, http.StatusConflict, "base_unit_locked")
	})
}
