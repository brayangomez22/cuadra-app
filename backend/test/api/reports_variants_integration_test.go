//go:build integration

package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReportsByVariant sells a clothing store's shirt in two sizes and
// checks the products report adds it up by product with the detail by
// variant, and filters by the tenant's attributes.
func TestReportsByVariant(t *testing.T) {
	f := &salesFlow{}
	f.handler, f.tokens = newAPI(t)
	rec := f.do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Almacén La Moda","nit":"900123456-8","name":"Luisa Ríos","email":"luisa@moda.co","password":"clave-segura-123","business_template":"ropa_calzado"}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
	}
	decode(t, rec, &session)
	owner := session.AccessToken

	rec = f.do(t, http.MethodPost, "/api/v1/products-with-variants",
		`{"name":"Camiseta básica","base_unit_id":"`+unitID(t, f.do, owner, "und")+`","tax_rate":"0.19","attributes":{"material":"algodón"},`+
			`"variants":[{"sku":"cam-M","cost":"20000","price":"40000","axis_values":{"talla":"M"}},`+
			`{"sku":"cam-L","cost":"20000","price":"40000","axis_values":{"talla":"L"}}]}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		Variants []struct{ ID, Label string }
	}
	decode(t, rec, &created)
	sizes := map[string]string{}
	for _, v := range created.Variants {
		sizes[v.Label] = v.ID
	}
	rec = f.do(t, http.MethodPost, "/api/v1/locations", `{"name":"Tienda"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.locationID = responseID(t, rec)
	rec = f.do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"0"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	for size, qty := range map[string]string{"M": "2", "L": "1"} {
		rec = f.do(t, http.MethodPost, "/api/v1/inventory/receipts",
			`{"variant_id":"`+sizes[size]+`","location_id":"`+f.locationID+`","quantity":"5","unit_cost":"20000"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodPost, "/api/v1/sales", `{"location_id":"`+f.locationID+`","lines":[{"variant_id":"`+
			sizes[size]+`","quantity":"`+qty+`"}],"payments":[{"method":"nequi","amount":"`+map[string]string{"2": "95200", "1": "47600"}[qty]+`"}]}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}
	today := time.Now().In(bogota).Format(time.DateOnly)
	period := "from=" + today + "&to=" + today
	r := &reportsFlow{salesFlow: f, period: period}

	t.Run("agrupa por producto con el detalle por talla, la que más vendió primero", func(t *testing.T) {
		var products productsReport
		r.report(t, "/api/v1/reports/products?"+period, owner, &products)
		require.Len(t, products.Items, 1)
		shirt := products.Items[0]
		require.Equal(t, "3", shirt.Quantity)
		require.Equal(t, "120000.00", shirt.Revenue)
		require.Equal(t, "60000.00", *shirt.Cost)
		require.Len(t, shirt.Variants, 2)
		require.Equal(t, "M", shirt.Variants[0].Label)
		require.Equal(t, "2", shirt.Variants[0].Quantity)
	})
	t.Run("filtra por un atributo de la empresa", func(t *testing.T) {
		var products productsReport
		r.report(t, "/api/v1/reports/products?attribute=material&attribute_value=algod%C3%B3n&"+period, owner, &products)
		require.Len(t, products.Items, 1)
		r.report(t, "/api/v1/reports/products?attribute=material&attribute_value=lino&"+period, owner, &products)
		require.Empty(t, products.Items)
	})
	t.Run("por medio de pago, Nequi suma las dos ventas", func(t *testing.T) {
		var sales salesReport
		r.report(t, "/api/v1/reports/sales?group_by=payment_method&"+period, owner, &sales)
		require.Equal(t, "nequi", sales.Items[0].Key)
		require.Equal(t, "142800.00", sales.Items[0].Total)
	})
}
