//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSalesVisibility checks P10 through the whole stack: a cashier sees
// only their own sales, the owner and the admin see everyone's, and the
// detail names who sold.
func TestSalesVisibility(t *testing.T) {
	f := newSalesFlow(t)
	do, owner := f.do, f.ownerToken
	other := joinStaff(t, do, owner, "cashier")
	admin := joinStaff(t, do, owner, "admin")

	sell := func(t *testing.T, token string) string {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"0"}`, token)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), token)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		return responseID(t, rec)
	}
	mine, theirs := sell(t, f.cashierToken), sell(t, other.Token)
	total := func(t *testing.T, token string) int {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/sales", "", token)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct{ Total int }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		return page.Total
	}

	t.Run("el cajero solo ve sus ventas del día", func(t *testing.T) {
		require.Equal(t, 1, total(t, f.cashierToken))
		require.Equal(t, http.StatusOK, do(t, http.MethodGet, "/api/v1/sales/"+mine, "", f.cashierToken).Code)
		rec := do(t, http.MethodGet, "/api/v1/sales/"+theirs, "", f.cashierToken)
		requireErrorCode(t, rec, http.StatusNotFound, "sale_not_found")
	})

	t.Run("el dueño y el administrador ven todas", func(t *testing.T) {
		require.Equal(t, 2, total(t, owner))
		require.Equal(t, 2, total(t, admin.Token))
	})

	t.Run("el detalle trae el nombre de quien vendió", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/sales/"+theirs, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"user_name":"Equipo cashier"`)
	})
}
