package app_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestAgingSort(t *testing.T) {
	// A hardware store's customer owes a lot, not yet due; a clothing
	// store's owes little, long overdue; a third owes some, a bit overdue.
	setUp := func(t *testing.T) (e *env, big, old, recent string) {
		t.Helper()
		e = newEnv(t)
		bigID := e.creditCustomer(t, "Constructora Andina", "9000000", 30)
		oldID := e.creditCustomer(t, "Boutique La Moda", "9000000", 0)
		recentID := e.creditCustomer(t, "Cliente con poco vencido", "9000000", 0)
		e.now = testNow.AddDate(0, 0, -120)
		e.charge(t, oldID, "50000") // 120 días vencida
		e.now = testNow.AddDate(0, 0, -5)
		e.charge(t, recentID, "80000") // 5 días vencida
		e.now = testNow
		e.charge(t, bigID, "2000000") // por vencer
		return e, bigID.String(), oldID.String(), recentID.String()
	}
	tests := []struct {
		name, sort string
		first      func(big, old, recent string) string
	}{
		{"por defecto, lo más vencido primero", "", func(_, _, recent string) string { return recent }},
		{"por lo que deben en total", "total", func(big, _, _ string) string { return big }},
		{"por lo vencido a más de 90 días", "over_90", func(_, old, _ string) string { return old }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, big, old, recent := setUp(t)
			report, err := e.svc.Aging(t.Context(), e.actor, app.AgingQuery{Sort: tt.sort, Limit: 10})
			require.NoError(t, err)
			require.Equal(t, tt.first(big, old, recent), report.Rows[0].CustomerID.String())
		})
	}
	t.Run("rechaza un orden desconocido", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.Aging(t.Context(), e.actor, app.AgingQuery{Sort: "name", Limit: 10})
		require.ErrorIs(t, err, domain.ErrInvalidAgingSort)
	})
}

func TestBalances(t *testing.T) {
	e := newEnv(t)
	owes := e.creditCustomer(t, "Ferretería cliente", "500000", 0)
	clean := e.creditCustomer(t, "Cliente al día", "500000", 30)
	e.now = testNow.AddDate(0, 0, -3)
	e.charge(t, owes, "30000") // 3 días vencida
	e.now = testNow
	e.charge(t, owes, "20000") // vence hoy

	balances, err := e.svc.Balances(t.Context(), e.actor, []uuid.UUID{owes, clean})

	require.NoError(t, err)
	require.True(t, dec("50000").Equal(balances[owes].Balance))
	require.True(t, dec("30000").Equal(balances[owes].Overdue))
	_, ok := balances[clean]
	require.False(t, ok, "un cliente sin cartera no aparece")
}
