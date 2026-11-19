//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// bogota is the tenants' time zone: the reports count days in Colombia.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// reportsFlow is a hardware store (salesFlow) whose cashier has a cash
// session open, and today's period for its reports.
type reportsFlow struct {
	*salesFlow
	sessionID, period string
}

func newReportsFlow(t *testing.T) *reportsFlow {
	t.Helper()
	f := &reportsFlow{salesFlow: newSalesFlow(t)}
	today := time.Now().In(bogota).Format(time.DateOnly)
	f.period = "from=" + today + "&to=" + today
	rec := f.do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"100000"}`, f.cashierToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.sessionID = responseID(t, rec)
	return f
}

// sell has the cashier sell qty cement bags paid with cash, and returns the
// sale's id.
func (f *reportsFlow) sell(t *testing.T, qty, cash string) string {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/v1/sales", f.saleBody(qty, cash), f.cashierToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return responseID(t, rec)
}

// receive has the owner receive qty cement bags at unitCost.
func (f *reportsFlow) receive(t *testing.T, variantID, qty, unitCost string) {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/v1/inventory/receipts",
		`{"variant_id":"`+variantID+`","location_id":"`+f.locationID+`","quantity":"`+qty+`","unit_cost":"`+unitCost+`"}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

// report reads a report as token into out.
func (f *reportsFlow) report(t *testing.T, path, token string, out any) {
	t.Helper()
	rec := f.do(t, http.MethodGet, path, "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
}

// salesReport is the body of GET /reports/sales.
type salesReport struct {
	Items []struct {
		Key, Label string
		SalesCount int `json:"sales_count"`
		Net, Tax   *string
		Total      string
	}
	Totals struct {
		SalesCount      int `json:"sales_count"`
		Net, Tax, Total string
	}
}

// productsReport is the body of GET /reports/products.
type productsReport struct {
	Items []struct {
		ProductID string `json:"product_id"`
		Name      string
		Quantity  string
		Revenue   string
		Cost      *string
		Profit    *string
		Margin    *string
		Variants  []struct {
			Label, Quantity, Revenue string
			Cost                     *string
		}
	}
	Total int
}

// cashReport is the body of GET /reports/cash-closings.
type cashReport struct {
	Items []struct {
		Key, Label     string
		Closings       int
		WithDifference int    `json:"with_difference"`
		SalesTotal     string `json:"sales_total"`
		Shortage       string
	}
	Totals struct {
		Closings   int
		SalesTotal string `json:"sales_total"`
		Shortage   string
		Difference string
	}
}

// teamReport is the body of GET /reports/team-control.
type teamReport struct {
	Users []struct {
		UserID           string `json:"user_id"`
		Name             string
		Voids            int
		VoidsTotal       string `json:"voids_total"`
		Adjustments      int
		AdjustmentsValue *string `json:"adjustments_value"`
		CashDifferences  int     `json:"cash_differences"`
		CashShortage     string  `json:"cash_shortage"`
	}
	Series []struct {
		UserID      string `json:"user_id"`
		BucketStart string `json:"bucket_start"`
		Voids       int
	}
}

// decode reads a response body into out.
func decode(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
}
