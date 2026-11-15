//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestInvoicingProvider prepares a tenant for the real provider (T36b): the
// DANE municipality of the company that issues the documents, its account
// at the provider with the token that never comes back, and the files of
// its documents.
func TestInvoicingProvider(t *testing.T) {
	f := &salesFlow{}
	startJobs(t, f)
	do, owner := f.do, f.ownerToken
	disableCashSession(t, f)

	t.Run("la empresa registra su municipio DANE y /me lo muestra", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/tenant", `{"city":"Medellín","municipality_code":"05001"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"municipality_code":"05001"`)
		rec = do(t, http.MethodGet, "/api/v1/me", "", owner)
		require.Contains(t, rec.Body.String(), `"tenant_municipality_code":"05001"`)

		rec = do(t, http.MethodPatch, "/api/v1/tenant", `{"municipality_code":"99999"}`, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_municipality")
	})

	t.Run("solo el dueño ve y cambia la cuenta en el proveedor", func(t *testing.T) {
		admin, _, err := f.tokens.Issue(f.tenantID, uuid.Must(uuid.NewV7()), "admin")
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/invoicing/provider", "", admin).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodPut, "/api/v1/invoicing/provider", `{"mode":"test"}`, admin).Code)
	})

	t.Run("el token se guarda y nunca vuelve", func(t *testing.T) {
		const token = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"
		rec := do(t, http.MethodGet, "/api/v1/invoicing/provider", "", owner)
		require.JSONEq(t, `{"mode":"test","test_set_id":"","has_token":false}`, rec.Body.String())

		rec = do(t, http.MethodPut, "/api/v1/invoicing/provider",
			`{"mode":"habilitation","test_set_id":"set-123","token":"`+token+`"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotContains(t, rec.Body.String(), token)

		rec = do(t, http.MethodGet, "/api/v1/invoicing/provider", "", owner)
		require.JSONEq(t, `{"mode":"habilitation","test_set_id":"set-123","has_token":true}`, rec.Body.String())

		rec = do(t, http.MethodPut, "/api/v1/invoicing/provider", `{"mode":"habilitation"}`, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "missing_test_set")
	})

	t.Run("un documento sin archivos del proveedor responde que no los tiene", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/invoicing/settings", `{"enabled":true,"default_kind":"pos"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/invoicing/resolutions", `{"kind":"pos","prefix":"pos","number":"18764000001234",`+
			`"range_from":1,"range_to":100,"valid_from":"2026-01-01","valid_to":"`+
			time.Now().AddDate(1, 0, 0).Format(time.DateOnly)+`"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var sale struct{ ID string }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sale))

		var docID string
		require.Eventually(t, func() bool {
			doc := documentsOf(t, f, sale.ID).Items[0]
			docID = doc.ID
			return doc.Status == "accepted"
		}, 15*time.Second, 100*time.Millisecond)

		rec = do(t, http.MethodGet, "/api/v1/invoicing/documents/"+docID+"/files/pdf", "", f.cashierToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "document_file_not_found")
	})
}
