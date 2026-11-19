//go:build integration

package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReportsAccess checks who reads the reports, the costs each one sees
// and that a tenant never sees another's.
func TestReportsAccess(t *testing.T) {
	f := newReportsFlow(t)
	f.sell(t, "1", "35700")

	t.Run("el cajero y el bodeguero no ven los informes del negocio", func(t *testing.T) {
		for _, token := range []string{f.cashierToken, f.warehouseToken} {
			rec := f.do(t, http.MethodGet, "/api/v1/reports/sales?"+f.period, "", token)
			require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		}
	})
	t.Run("el administrador ve el reporte de productos con su costo", func(t *testing.T) {
		admin := joinStaff(t, f.do, f.ownerToken, "admin").Token
		var r productsReport
		f.report(t, "/api/v1/reports/products?"+f.period, admin, &r)
		require.NotNil(t, r.Items[0].Cost)
	})
	t.Run("un periodo invertido responde 422 invalid_period", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/reports/sales?from=2026-11-04&to=2026-11-03", "", f.ownerToken)
		requireErrorCode(t, rec, http.StatusUnprocessableEntity, "invalid_period")
	})
	t.Run("aislamiento: otra empresa no ve las ventas, los productos ni el control de la primera", func(t *testing.T) {
		other, _ := f.signup(t, "otra@empresa.co")
		var sales salesReport
		f.report(t, "/api/v1/reports/sales?group_by=user&"+f.period, other, &sales)
		require.Zero(t, sales.Totals.SalesCount)
		require.Empty(t, sales.Items)
		var products productsReport
		f.report(t, "/api/v1/reports/products?"+f.period, other, &products)
		require.Empty(t, products.Items)
		var team teamReport
		f.report(t, "/api/v1/reports/team-control?"+f.period, other, &team)
		require.Empty(t, team.Users)
		var cash cashReport
		f.report(t, "/api/v1/reports/cash-closings?"+f.period, other, &cash)
		require.Empty(t, cash.Items)
		var idle idleReport
		f.report(t, "/api/v1/reports/idle-stock", other, &idle)
		require.Empty(t, idle.Items)
	})
}

// idleReport is the body of GET /reports/idle-stock.
type idleReport struct {
	Items []struct {
		Name       string
		Quantity   string
		Value      *string
		LastSoldAt *string `json:"last_sold_at"`
	}
	Value *string
}

// adjustmentsReport is the body of GET /reports/stock-adjustments.
type adjustmentsReport struct {
	Items []struct {
		Key, Label  string
		Adjustments int
		QuantityOut string `json:"quantity_out"`
		Value       *string
	}
	Adjustments int
	Value       *string
}

// TestReportsStock checks the idle stock and the stock adjustments.
func TestReportsStock(t *testing.T) {
	f := newReportsFlow(t)
	rec := f.do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"tor-14","name":"Tornillo 1/4","base_unit_id":"`+unitID(t, f.do, f.ownerToken, "und")+
			`","cost":"150","price":"500","tax_rate":"0.19","attributes":{}}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var screw struct {
		Variants []struct{ ID string } `json:"variants"`
	}
	decode(t, rec, &screw)
	f.receive(t, screw.Variants[0].ID, "200", "150")
	f.sell(t, "1", "35700")
	rec = f.do(t, http.MethodPost, "/api/v1/inventory/adjustments",
		`{"variant_id":"`+f.variantID+`","location_id":"`+f.locationID+`","quantity":"-1","reason":"Bulto roto"}`, f.warehouseToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	t.Run("sin movimiento: el tornillo que nunca se vendió, valorizado al costo; el cemento no", func(t *testing.T) {
		var idle idleReport
		f.report(t, "/api/v1/reports/idle-stock?days=30&location_id="+f.locationID, f.ownerToken, &idle)
		require.Len(t, idle.Items, 1)
		require.Equal(t, "Tornillo 1/4", idle.Items[0].Name)
		require.Equal(t, "200", idle.Items[0].Quantity)
		require.Equal(t, "30000.00", *idle.Items[0].Value)
		require.Nil(t, idle.Items[0].LastSoldAt)
	})
	t.Run("los ajustes por variante salen valorizados al costo del momento", func(t *testing.T) {
		var adj adjustmentsReport
		f.report(t, "/api/v1/reports/stock-adjustments?group_by=variant&"+f.period, f.ownerToken, &adj)
		require.Len(t, adj.Items, 1)
		require.Equal(t, f.variantID, adj.Items[0].Key)
		require.Equal(t, "1", adj.Items[0].QuantityOut)
		require.Equal(t, "-25000.00", *adj.Items[0].Value)
		require.Equal(t, 1, adj.Adjustments)

		var team teamReport
		f.report(t, "/api/v1/reports/team-control?"+f.period, f.ownerToken, &team)
		adjusted := 0
		for _, u := range team.Users {
			adjusted += u.Adjustments
		}
		require.Equal(t, 1, adjusted)
	})
}
