package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

var (
	tenantID   = uuid.Must(uuid.NewV7())
	customerID = uuid.Must(uuid.NewV7())
	// soldAt is 2026-10-28 at 10:00 in Bogotá.
	soldAt = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func terms(t *testing.T, limit string, days int) domain.Terms {
	t.Helper()
	tr, err := domain.NewTerms(dec(limit), days)
	require.NoError(t, err)
	return tr
}

func newAccount(t *testing.T, limit string, days int) *domain.Account {
	t.Helper()
	a, err := domain.NewAccount(tenantID, customerID, terms(t, limit, days))
	require.NoError(t, err)
	return a
}

func charge(t *testing.T, a *domain.Account, amount string, at time.Time) *domain.Receivable {
	t.Helper()
	r, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: newID(), Amount: dec(amount), At: at})
	require.NoError(t, err)
	return r
}

func TestCreditTerms(t *testing.T) {
	t.Run("acepta cupos y plazos de distintos tipos de negocio", func(t *testing.T) {
		cases := []struct {
			name  string
			limit string
			days  int
		}{
			{"cliente de ferretería con cupo y 30 días", "2000000", 30},
			{"cliente de almacén de ropa con 15 días", "350000.50", 15},
			{"cupo cero desactiva el crédito", "0", 0},
			{"plazo máximo de un año", "100000", 365},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				tr, err := domain.NewTerms(dec(c.limit), c.days)
				require.NoError(t, err)
				require.True(t, dec(c.limit).Equal(tr.Limit()))
				require.Equal(t, c.days, tr.Days())
			})
		}
	})

	t.Run("rechaza cupo negativo o con más de dos decimales", func(t *testing.T) {
		for _, limit := range []string{"-1", "100.001"} {
			_, err := domain.NewTerms(dec(limit), 30)
			require.ErrorIs(t, err, domain.ErrInvalidCreditLimit)
		}
	})

	t.Run("rechaza plazo negativo o de más de un año", func(t *testing.T) {
		for _, days := range []int{-1, 366} {
			_, err := domain.NewTerms(dec("1000"), days)
			require.ErrorIs(t, err, domain.ErrInvalidCreditTerm)
		}
	})

	t.Run("rechaza cuenta sin tenant o sin cliente", func(t *testing.T) {
		_, err := domain.NewAccount(uuid.Nil, customerID, terms(t, "1000", 30))
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
		_, err = domain.NewAccount(tenantID, uuid.Nil, terms(t, "1000", 30))
		require.ErrorIs(t, err, domain.ErrInvalidCustomerID)
	})

	t.Run("una cuenta nueva no debe nada y tiene todo el cupo disponible", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		require.True(t, a.Balance().IsZero())
		require.True(t, dec("500000").Equal(a.Available()))
	})

	t.Run("bajar el cupo por debajo del saldo deja el disponible en cero", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		charge(t, a, "400000", soldAt)
		a.SetTerms(terms(t, "300000", 30))
		require.True(t, a.Available().IsZero())
		require.True(t, dec("400000").Equal(a.Balance()))
	})
}

func TestAccountCharge(t *testing.T) {
	t.Run("una venta a crédito dentro del cupo abre la cuenta por cobrar", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		saleID := newID()
		r, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: saleID, Amount: dec("120000"), At: soldAt})
		require.NoError(t, err)
		require.Equal(t, saleID, r.SaleID())
		require.Equal(t, customerID, r.CustomerID())
		require.True(t, dec("120000").Equal(r.Amount()))
		require.True(t, dec("120000").Equal(r.Balance()))
		require.Equal(t, domain.ReceivableOpen, r.Status())
		require.Nil(t, r.Authorization())
		require.True(t, dec("120000").Equal(a.Balance()))
		require.True(t, dec("380000").Equal(a.Available()))
	})

	t.Run("una venta por exactamente el disponible se acepta", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		charge(t, a, "500000", soldAt)
		require.True(t, a.Available().IsZero())
	})

	t.Run("rechaza venta que excede el cupo sin autorización e informa el disponible", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		charge(t, a, "450000", soldAt)
		_, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: newID(), Amount: dec("50000.01"), At: soldAt})
		require.ErrorIs(t, err, domain.ErrCreditLimitExceeded)
		var exceeded *domain.CreditLimitExceededError
		require.ErrorAs(t, err, &exceeded)
		require.True(t, dec("50000").Equal(exceeded.Available))
		require.True(t, dec("450000").Equal(a.Balance()), "el saldo no cambia")
	})

	t.Run("acepta venta que excede el cupo con autorización y guarda quién autorizó", func(t *testing.T) {
		a := newAccount(t, "100000", 30)
		authorizer := newID()
		r, err := a.Charge(domain.ChargeInput{
			ID: newID(), SaleID: newID(), Amount: dec("150000"), At: soldAt,
			Authorization: &domain.Authorization{UserID: authorizer, At: soldAt},
		})
		require.NoError(t, err)
		require.NotNil(t, r.Authorization())
		require.Equal(t, authorizer, r.Authorization().UserID)
		require.True(t, dec("150000").Equal(a.Balance()))
		require.True(t, a.Available().IsZero())
	})

	t.Run("una autorización que no hizo falta no se guarda", func(t *testing.T) {
		a := newAccount(t, "100000", 30)
		r, err := a.Charge(domain.ChargeInput{
			ID: newID(), SaleID: newID(), Amount: dec("50000"), At: soldAt,
			Authorization: &domain.Authorization{UserID: newID(), At: soldAt},
		})
		require.NoError(t, err)
		require.Nil(t, r.Authorization())
	})

	t.Run("rechaza venta a crédito de un cliente con cupo cero aun con autorización", func(t *testing.T) {
		a := newAccount(t, "0", 30)
		_, err := a.Charge(domain.ChargeInput{
			ID: newID(), SaleID: newID(), Amount: dec("1000"), At: soldAt,
			Authorization: &domain.Authorization{UserID: newID(), At: soldAt},
		})
		require.ErrorIs(t, err, domain.ErrCreditNotEnabled)
	})

	t.Run("rechaza cargo de monto cero, negativo o con más de dos decimales", func(t *testing.T) {
		a := newAccount(t, "100000", 30)
		for _, amount := range []string{"0", "-5", "10.005"} {
			_, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: newID(), Amount: dec(amount), At: soldAt})
			require.ErrorIs(t, err, domain.ErrInvalidChargeAmount)
		}
	})

	t.Run("calcula el vencimiento con la fecha de venta en America/Bogota", func(t *testing.T) {
		cases := []struct {
			name string
			at   time.Time
			days int
			due  domain.Date
		}{
			{"ferretería a 30 días", soldAt, 30, domain.NewDate(2026, 11, 27)},
			// 2026-10-29 02:00 UTC is still 2026-10-28 at 21:00 in Bogotá.
			{"almacén de ropa de noche, día de Bogotá y no de UTC", time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC), 15, domain.NewDate(2026, 11, 12)},
			{"plazo cero vence el mismo día", soldAt, 0, domain.NewDate(2026, 10, 28)},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				a := newAccount(t, "1000000", c.days)
				r := charge(t, a, "1000", c.at)
				require.Equal(t, c.due, r.DueDate())
			})
		}
	})
}

func TestAccountCancel(t *testing.T) {
	t.Run("anular una venta a crédito sin abonos cancela su cuenta y libera el cupo", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		require.NoError(t, a.Cancel(r, soldAt))
		require.Equal(t, domain.ReceivableCanceled, r.Status())
		require.True(t, r.Balance().IsZero())
		require.True(t, a.Balance().IsZero())
		require.True(t, dec("500000").Equal(a.Available()))
	})

	t.Run("rechaza cancelar una cuenta con abonos", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		_, err := a.Collect(paymentInput("50000"), []*domain.Receivable{r})
		require.NoError(t, err)
		require.ErrorIs(t, a.Cancel(r, soldAt), domain.ErrReceivableHasPayments)
		require.True(t, dec("150000").Equal(a.Balance()))
	})

	t.Run("rechaza cancelar dos veces", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		require.NoError(t, a.Cancel(r, soldAt))
		require.ErrorIs(t, a.Cancel(r, soldAt), domain.ErrReceivableNotOpen)
	})
}
