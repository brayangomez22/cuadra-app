package simulated_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/simulated"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestInvoicer(t *testing.T) {
	ref := app.DocumentRef{TenantID: uuid.Must(uuid.NewV7()), Kind: domain.KindPOS, Prefix: "POS", Number: 41}

	t.Run("acepta el documento con un CUDE estable por número y un QR que dice que es simulado", func(t *testing.T) {
		inv := simulated.Invoicer{}
		first, err := inv.Issue(t.Context(), app.IssueRequest{DocumentRef: ref})
		require.NoError(t, err)
		again, err := inv.Issue(t.Context(), app.IssueRequest{DocumentRef: ref})
		require.NoError(t, err)

		require.Equal(t, domain.StatusAccepted, first.Status)
		require.Len(t, first.Code, 96, "un CUDE es un SHA-384 en hexadecimal")
		require.Equal(t, first, again)
		require.Contains(t, first.QR, "simulado")
	})

	t.Run("otro número da otro CUDE", func(t *testing.T) {
		inv := simulated.Invoicer{}
		a, err := inv.Issue(t.Context(), app.IssueRequest{DocumentRef: ref})
		require.NoError(t, err)
		other := ref
		other.Number++
		b, err := inv.Issue(t.Context(), app.IssueRequest{DocumentRef: other})
		require.NoError(t, err)
		require.NotEqual(t, a.Code, b.Code)
	})

	t.Run("la consulta encuentra lo que acepta, porque responde siempre igual", func(t *testing.T) {
		inv := simulated.Invoicer{}
		issued, err := inv.Issue(t.Context(), app.IssueRequest{DocumentRef: ref})
		require.NoError(t, err)
		found, ok, err := inv.Lookup(t.Context(), ref)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, issued, found)
	})
}
