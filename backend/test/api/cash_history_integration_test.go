//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestCashHistory lists the cash sessions with their cashier's name and lets
// who manages cash close the one another user left open.
func TestCashHistory(t *testing.T) {
	f := newSalesFlow(t)
	do := f.do
	admin, _, err := f.tokens.Issue(f.tenantID, uuid.Must(uuid.NewV7()), "admin")
	require.NoError(t, err)

	// The owner (Ana Gómez, from the sign-up) opens a drawer and leaves it open.
	rec := do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"50000"}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	sessionID := responseID(t, rec)

	t.Run("el cajero y el bodeguero no ven el historial", func(t *testing.T) {
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/cash-sessions", "", f.cashierToken).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/cash-sessions", "", f.warehouseToken).Code)
	})

	t.Run("el administrador ve la caja abierta con el nombre de su cajero", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/cash-sessions?status=open", "", admin)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct {
				ID          string `json:"id"`
				CashierName string `json:"cashier_name"`
				Status      string `json:"status"`
			}
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 1, page.Total)
		require.Equal(t, sessionID, page.Items[0].ID)
		require.Equal(t, "Ana Gómez", page.Items[0].CashierName)
	})

	t.Run("el administrador cierra la caja que otro dejó abierta y queda como quien la cerró", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/close",
			`{"counts":[{"denomination":"50000","quantity":1}],"notes":"La dejó abierta"}`, admin)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"result":"exact"`)
		require.Contains(t, rec.Body.String(), `"sales_total":"0.00"`)

		rec = do(t, http.MethodGet, "/api/v1/cash-sessions?status=closed", "", admin)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"result":"exact"`)
	})
}
