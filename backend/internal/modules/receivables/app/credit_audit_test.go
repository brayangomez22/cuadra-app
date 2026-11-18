package app_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
)

func TestCreditTermsAudit(t *testing.T) {
	cases := []struct {
		name, limit string
		days        int
	}{
		{"cliente de ferretería", "3000000", 30},
		{"clienta de almacén de ropa", "400000", 15},
	}
	for _, c := range cases {
		t.Run("el cambio de cupo y plazo de un "+c.name+" queda en la auditoría", func(t *testing.T) {
			e := newEnv(t)
			id := e.creditCustomer(t, c.name, "100000", 10)
			e.audit.entries = nil

			_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec(c.limit), Days: c.days})
			require.NoError(t, err)
			require.Len(t, e.audit.entries, 1)
			require.Equal(t, app.AuditEntry{
				Action: "credit.terms_changed", EntityID: id,
				Details: map[string]string{
					"old_limit": "100000.00", "new_limit": dec(c.limit).StringFixed(2),
					"old_term_days": "10", "new_term_days": strconv.Itoa(c.days),
				},
			}, e.audit.entries[0])
		})
	}

	t.Run("sin cambios no hay evento", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "100000", 10)
		e.audit.entries = nil
		_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec("100000"), Days: 10})
		require.NoError(t, err)
		require.Empty(t, e.audit.entries)
	})

	t.Run("si la auditoría falla el cambio de cupo falla (y la transacción se deshace)", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "100000", 10)
		e.audit.fail = true
		_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec("500000"), Days: 30})
		require.ErrorIs(t, err, errAudit)
	})
}
