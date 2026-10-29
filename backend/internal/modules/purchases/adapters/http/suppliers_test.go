package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestSupplierHandlers(t *testing.T) {
	t.Run("crea un proveedor activo por defecto y responde su saldo como string", func(t *testing.T) {
		uc := &stubUseCases{supplier: aSupplier(t, "0")}
		rec := serve(t, uc, http.MethodPost, "/api/v1/suppliers",
			`{"name":"Distribuidora del Valle","nit":"900.123.456-8","contact_name":"Laura Pérez","payment_term_days":30}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, domain.SupplierParams{
			Name: "Distribuidora del Valle", NIT: "900.123.456-8", Contact: domain.Contact{Name: "Laura Pérez"},
			PaymentTermDays: 30, Active: true,
		}, uc.params)
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "900123456-8", got["nit"])
		require.Equal(t, "0.00", got["balance_due"])
		require.Equal(t, "ventas@valle.co", got["email"])
	})

	t.Run("lista con el filtro del query y el saldo de cada proveedor", func(t *testing.T) {
		uc := &stubUseCases{suppliers: app.SupplierPage{Items: []app.SupplierView{aSupplier(t, "107100.5")}, Total: 1}}
		rec := serve(t, uc, http.MethodGet, "/api/v1/suppliers?q=valle&status=all&limit=5&offset=10", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.SupplierQuery{Text: "valle", Status: domain.SuppliersAll, Limit: 5, Offset: 10}, uc.supplierQuery)
		require.Contains(t, rec.Body.String(), `"balance_due":"107100.50"`)
	})

	t.Run("mapea los errores del proveedor", func(t *testing.T) {
		for _, tc := range []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrNITTaken, http.StatusConflict, "supplier_nit_taken"},
			{domain.ErrInvalidNITCheckDigit, http.StatusUnprocessableEntity, "invalid_nit_check_digit"},
			{domain.ErrInvalidNIT, http.StatusUnprocessableEntity, "invalid_nit"},
			{domain.ErrInvalidPaymentTerm, http.StatusUnprocessableEntity, "invalid_payment_term"},
			{domain.ErrSupplierNotFound, http.StatusNotFound, "supplier_not_found"},
		} {
			uc := &stubUseCases{err: tc.err}
			rec := serve(t, uc, http.MethodPut, "/api/v1/suppliers/"+newID().String(), `{"name":"X","nit":"900123456-8"}`)
			requireError(t, rec, tc.status, tc.code)
		}
	})
}
