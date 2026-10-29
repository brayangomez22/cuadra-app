package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestStatement(t *testing.T) {
	t.Run("el estado de cuenta trae la cuenta y una página de movimientos", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "100000")
		e.charge(t, id, "20000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "nequi", Amount: dec("30000")})
		require.NoError(t, err)
		page, err := e.svc.Statement(t.Context(), e.actor, id, 2, 0)
		require.NoError(t, err)
		require.Equal(t, 3, page.Total)
		require.Len(t, page.Movements, 2)
		require.Equal(t, domain.MovementPayment, page.Movements[0].Kind)
		require.True(t, dec("90000").Equal(page.Movements[0].Balance))
		require.True(t, dec("90000").Equal(page.Account.Balance()))
		require.Equal(t, id.String(), e.span(t, "receivables.Statement")["customer.id"].AsString())
	})

	t.Run("rechaza el estado de cuenta de un cliente que no existe", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.Statement(t.Context(), e.actor, newID(), 10, 0)
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}

func TestAging(t *testing.T) {
	t.Run("agrupa la cartera por cliente con su nombre, lo más vencido primero y con el día de Bogotá", func(t *testing.T) {
		e := newEnv(t)
		hardware := e.creditCustomer(t, "Cliente de ferretería", "5000000", 30)
		clothing := e.creditCustomer(t, "Clienta de almacén de ropa", "5000000", 0)
		upToDate := e.creditCustomer(t, "Cliente al día", "5000000", 30)

		e.now = testNow.AddDate(0, 0, -100)
		e.charge(t, hardware, "100000") // 70 días vencida
		e.now = testNow.AddDate(0, 0, -10)
		e.charge(t, clothing, "40000") // 10 días vencida
		e.now = testNow
		e.charge(t, clothing, "5000") // vence hoy en Bogotá
		e.charge(t, upToDate, "70000")

		report, err := e.svc.Aging(t.Context(), e.actor, 10, 0)
		require.NoError(t, err)
		require.Equal(t, domain.NewDate(2026, 10, 28), report.AsOf)
		require.Equal(t, 3, report.Total)
		require.Len(t, report.Rows, 3)

		first, second, third := report.Rows[0], report.Rows[1], report.Rows[2]
		require.Equal(t, hardware, first.CustomerID)
		require.Equal(t, "Cliente de ferretería", first.Name)
		require.True(t, dec("100000").Equal(first.Aging.Days61To90))
		require.Equal(t, clothing, second.CustomerID)
		require.True(t, dec("40000").Equal(second.Aging.Days1To30))
		require.True(t, dec("5000").Equal(second.Aging.Current))
		require.Equal(t, upToDate, third.CustomerID)
		require.True(t, third.Aging.Overdue().IsZero())

		require.True(t, dec("215000").Equal(report.Totals.Total()))
		require.True(t, dec("140000").Equal(report.Totals.Overdue()))
		e.span(t, "receivables.Aging")
	})

	t.Run("pagina la cartera por edades sin perder los totales", func(t *testing.T) {
		e := newEnv(t)
		for _, name := range []string{"A", "B", "C"} {
			e.charge(t, e.creditCustomer(t, name, "100000", 30), "1000")
		}
		report, err := e.svc.Aging(t.Context(), e.actor, 2, 2)
		require.NoError(t, err)
		require.Equal(t, 3, report.Total)
		require.Len(t, report.Rows, 1)
		require.True(t, dec("3000").Equal(report.Totals.Total()))
	})

	t.Run("sin cartera el informe sale vacío", func(t *testing.T) {
		e := newEnv(t)
		e.now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		report, err := e.svc.Aging(t.Context(), e.actor, 10, 0)
		require.NoError(t, err)
		require.Zero(t, report.Total)
		require.Empty(t, report.Rows)
		require.True(t, report.Totals.Total().IsZero())
	})
}
