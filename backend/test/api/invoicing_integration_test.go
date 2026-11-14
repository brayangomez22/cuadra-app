//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestInvoicingFlow issues the electronic documents of sales through the
// whole stack: emission off by default, the numbering resolution in force,
// the document numbered in the sale's transaction, its issue in the
// background by the simulated provider and the permissions of each role.
func TestInvoicingFlow(t *testing.T) {
	f := &salesFlow{}
	startJobs(t, f)
	do, owner, cashier := f.do, f.ownerToken, f.cashierToken
	disableCashSession(t, f)

	var resolutionID, saleID, documentID string

	t.Run("con la emisión apagada la venta no tiene documento", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Zero(t, documentsOf(t, f, responseID(t, rec)).Total)
	})

	t.Run("con la emisión activa y sin resolución vigente la venta no se confirma", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/invoicing/settings", `{"enabled":true,"default_kind":"pos"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "no_active_resolution")
		require.Equal(t, "9", f.stockOf(t, f.variantID), "no sale el stock")
	})

	t.Run("el dueño registra la resolución y el cajero no puede", func(t *testing.T) {
		body := `{"kind":"pos","prefix":"pos","number":"18764000001234","range_from":1,"range_to":10,` +
			`"valid_from":"2026-01-01","valid_to":"` + time.Now().AddDate(1, 0, 0).Format(time.DateOnly) + `"}`
		rec := do(t, http.MethodPost, "/api/v1/invoicing/resolutions", body, cashier)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/invoicing/resolutions", body, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		resolutionID = responseID(t, rec)
		require.Contains(t, rec.Body.String(), `"prefix":"POS"`)
		require.Contains(t, rec.Body.String(), `"next_number":1`)
		require.Contains(t, rec.Body.String(), `"near_end":false`)

		rec = do(t, http.MethodPost, "/api/v1/invoicing/resolutions", body, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "resolution_overlaps")
	})

	t.Run("la venta toma el número y el trabajo la emite en segundo plano", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		saleID = responseID(t, rec)
		page := documentsOf(t, f, saleID)
		require.Equal(t, 1, page.Total)
		doc := page.Items[0]
		documentID = doc.ID
		require.Equal(t, "pos", doc.Kind)
		require.Equal(t, "POS", doc.Prefix)
		require.Equal(t, int64(1), doc.Number)

		require.Eventually(t, func() bool {
			return documentsOf(t, f, saleID).Items[0].Status == "accepted"
		}, 15*time.Second, 100*time.Millisecond)
		doc = documentsOf(t, f, saleID).Items[0]
		require.Len(t, doc.Code, 96)
		require.Contains(t, doc.QR, "simulado")
	})

	t.Run("el cajero ve el documento pero no lo reenvía; uno aceptado no se reenvía", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/invoicing/documents/"+documentID, "", cashier)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/invoicing/documents/"+documentID+"/retry", "", cashier)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/invoicing/documents/"+documentID+"/retry", "", owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "document_not_retryable")
	})

	t.Run("no anula una venta con documento ni borra la resolución que la numeró", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"Error del cajero"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "sale_invoiced")
		rec = do(t, http.MethodDelete, "/api/v1/invoicing/resolutions/"+resolutionID, "", owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "resolution_in_use")
	})

	t.Run("factura pedida sin cliente se rechaza", func(t *testing.T) {
		body := `{"location_id":"` + f.locationID + `","document_kind":"invoice","lines":[{"variant_id":"` + f.variantID +
			`","quantity":"1"}],"payments":[{"method":"cash","amount":"35700"}]}`
		rec := do(t, http.MethodPost, "/api/v1/sales", body, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invoice_requires_customer")
	})

	t.Run("aislamiento: otra empresa no ve los documentos ni las resoluciones", func(t *testing.T) {
		other, _ := f.signup(t, "otra@facturas.co")
		rec := do(t, http.MethodGet, "/api/v1/invoicing/documents/"+documentID, "", other)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		rec = do(t, http.MethodGet, "/api/v1/invoicing/resolutions", "", other)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `[]`, rec.Body.String())
	})
}

// startJobs builds the API with its background jobs running and sets f up.
func startJobs(t *testing.T, f *salesFlow) {
	t.Helper()
	handler, tokens, jobClient := newAPIWithJobs(t)
	f.handler, f.tokens = handler, tokens
	f.setUp(t)
	require.NoError(t, jobClient.Start(t.Context()))
	t.Cleanup(func() { _ = jobClient.Stop(t.Context()) })
}

// disableCashSession lets the tenant sell without an open cash session.
func disableCashSession(t *testing.T, f *salesFlow) {
	t.Helper()
	rec := f.do(t, http.MethodPatch, "/api/v1/sales/settings", `{"require_cash_session":false}`, f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

type documentPage struct {
	Total int
	Items []struct {
		ID, Kind, Prefix, Status, Code string
		QR                             string `json:"qr"`
		Number                         int64
	}
}

// documentsOf returns the documents of a sale.
func documentsOf(t *testing.T, f *salesFlow, saleID string) documentPage {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/invoicing/documents?sale_id="+saleID, "", f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var page documentPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	return page
}
