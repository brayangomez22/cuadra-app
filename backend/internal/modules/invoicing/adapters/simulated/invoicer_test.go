package simulated_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/simulated"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestInvoicer(t *testing.T) {
	ref := app.DocumentRef{TenantID: uuid.Must(uuid.NewV7()), Kind: domain.KindPOS, Prefix: "POS", Number: 41}

	t.Run("acepta el documento con un CUDE estable por número y un QR que dice que es simulado", func(t *testing.T) {
		inv := simulated.Invoicer{}
		first, err := inv.Issue(t.Context(), complete(ref))
		require.NoError(t, err)
		again, err := inv.Issue(t.Context(), complete(ref))
		require.NoError(t, err)

		require.Equal(t, domain.StatusAccepted, first.Status)
		require.Len(t, first.Code, 96, "un CUDE es un SHA-384 en hexadecimal")
		require.Equal(t, first, again)
		require.Contains(t, first.QR, "simulado")
	})

	t.Run("otro número da otro CUDE", func(t *testing.T) {
		inv := simulated.Invoicer{}
		a, err := inv.Issue(t.Context(), complete(ref))
		require.NoError(t, err)
		other := ref
		other.Number++
		b, err := inv.Issue(t.Context(), complete(other))
		require.NoError(t, err)
		require.NotEqual(t, a.Code, b.Code)
	})

	t.Run("la consulta encuentra lo que acepta, porque responde siempre igual", func(t *testing.T) {
		inv := simulated.Invoicer{}
		issued, err := inv.Issue(t.Context(), complete(ref))
		require.NoError(t, err)
		found, ok, err := inv.Lookup(t.Context(), ref)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, issued, found)
	})
}

// complete is a request with every field the provider requires.
func complete(ref app.DocumentRef) app.IssueRequest {
	return app.IssueRequest{
		DocumentRef: ref,
		Buyer:       domain.FinalConsumer(),
		Sale: app.SaleData{Lines: []app.SaleLine{{
			Code: "CAB-12", UnitCode: "MTR", Name: "Cable #12", Quantity: decimal.NewFromInt(3),
			TaxTreatment: vat.Taxed, TaxRate: decimal.RequireFromString("0.19"),
		}}},
	}
}

func TestInvoicerRequiredData(t *testing.T) {
	ref := app.DocumentRef{TenantID: uuid.Must(uuid.NewV7()), Kind: domain.KindPOS, Prefix: "POS", Number: 7}
	tests := []struct {
		name   string
		mutate func(*app.IssueRequest)
	}{
		{"rechaza un documento sin líneas", func(r *app.IssueRequest) { r.Sale.Lines = nil }},
		{"rechaza una línea sin código", func(r *app.IssueRequest) { r.Sale.Lines[0].Code = "" }},
		{"rechaza una línea sin código de unidad", func(r *app.IssueRequest) { r.Sale.Lines[0].UnitCode = "" }},
		{"rechaza una línea sin tratamiento del IVA", func(r *app.IssueRequest) { r.Sale.Lines[0].TaxTreatment = "" }},
		{"rechaza un comprador sin código DIAN de su documento", func(r *app.IssueRequest) { r.Buyer.IdentificationCode = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := complete(ref)
			tt.mutate(&req)
			_, err := simulated.Invoicer{}.Issue(t.Context(), req)
			var perr *app.ProviderError
			require.ErrorAs(t, err, &perr)
			require.False(t, perr.Temporary(), "una petición incompleta no se arregla reintentando")
		})
	}
}
