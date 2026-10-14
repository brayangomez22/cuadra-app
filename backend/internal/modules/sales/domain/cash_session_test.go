package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func cashParams(float string) domain.NewCashSessionParams {
	return domain.NewCashSessionParams{
		ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(),
		OpeningFloat: dec(float), OpenedAt: testNow,
	}
}

func openSession(t *testing.T, float string) *domain.CashSession {
	t.Helper()
	s, err := domain.OpenCashSession(cashParams(float))
	require.NoError(t, err)
	return s
}

func movement(s *domain.CashSession, kind domain.CashMovementKind, amount, reason string) domain.CashMovementInput {
	return domain.CashMovementInput{
		ID: newID(), Kind: kind, Amount: dec(amount), Reason: reason, UserID: s.UserID(), At: testNow.Add(time.Hour),
	}
}

func count(t *testing.T, pairs ...any) domain.CashCount {
	t.Helper()
	var counts []domain.DenominationCount
	for i := 0; i < len(pairs); i += 2 {
		counts = append(counts, domain.DenominationCount{Denomination: dec(pairs[i].(string)), Quantity: int64(pairs[i+1].(int))})
	}
	c, err := domain.NewCashCount(counts)
	require.NoError(t, err)
	return c
}

func activity(received, change, in, out string) domain.CashActivity {
	return domain.CashActivity{
		CashReceived: dec(received), ChangeGiven: dec(change), CashIn: dec(in), CashOut: dec(out),
	}
}

func closeInput(s *domain.CashSession, c domain.CashCount, a domain.CashActivity) domain.CloseCashInput {
	return domain.CloseCashInput{UserID: s.UserID(), Count: c, Activity: a, At: testNow.Add(8 * time.Hour)}
}

func TestOpenCashSession(t *testing.T) {
	t.Run("abre la caja con base inicial cero o positiva", func(t *testing.T) {
		for _, float := range []string{"0", "150000", "150000.50"} {
			p := cashParams(float)
			s, err := domain.OpenCashSession(p)
			require.NoError(t, err, float)
			require.Equal(t, domain.CashSessionOpen, s.Status())
			require.Equal(t, p.ID, s.ID())
			require.Equal(t, p.TenantID, s.TenantID())
			require.Equal(t, p.LocationID, s.LocationID())
			require.Equal(t, p.UserID, s.UserID())
			requireDec(t, float, s.OpeningFloat())
			require.Equal(t, testNow.UTC(), s.OpenedAt())
			require.Nil(t, s.Closing())
		}
	})

	t.Run("rechaza una base negativa o con más de 2 decimales", func(t *testing.T) {
		for _, float := range []string{"-1", "100.005", "100000000000000"} {
			_, err := domain.OpenCashSession(cashParams(float))
			require.ErrorIs(t, err, domain.ErrInvalidOpeningFloat, float)
		}
	})

	t.Run("rechaza la caja sin id, tenant, sede o usuario", func(t *testing.T) {
		tests := []struct {
			name   string
			change func(*domain.NewCashSessionParams)
			want   error
		}{
			{"sin id", func(p *domain.NewCashSessionParams) { p.ID = uuid.Nil }, domain.ErrInvalidCashSessionID},
			{"sin tenant", func(p *domain.NewCashSessionParams) { p.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"sin sede", func(p *domain.NewCashSessionParams) { p.LocationID = uuid.Nil }, domain.ErrInvalidLocationID},
			{"sin usuario", func(p *domain.NewCashSessionParams) { p.UserID = uuid.Nil }, domain.ErrInvalidUserID},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := cashParams("0")
				tt.change(&p)
				_, err := domain.OpenCashSession(p)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}

func TestExpectedCash(t *testing.T) {
	tests := []struct {
		name                    string
		float, received, change string
		in, out                 string
		want                    string
	}{
		{name: "sin ventas ni movimientos el esperado es la base", float: "100000", received: "0", change: "0", in: "0", out: "0", want: "100000"},
		{name: "suma el efectivo recibido y resta el cambio entregado", float: "100000", received: "250000", change: "12300.50", in: "0", out: "0", want: "337699.50"},
		{name: "suma los ingresos y resta los egresos", float: "50000", received: "0", change: "0", in: "20000", out: "35000", want: "35000"},
		{name: "el esperado en caja = base + ventas en efectivo + ingresos − egresos − cambio entregado", float: "100000", received: "480000", change: "30000", in: "50000", out: "120000", want: "480000"},
		{name: "puede quedar negativo si los egresos superan lo que hay", float: "0", received: "0", change: "0", in: "0", out: "10000", want: "-10000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ExpectedCash(dec(tt.float), activity(tt.received, tt.change, tt.in, tt.out))
			requireDec(t, tt.want, got)
		})
	}
}

func TestCashSessionRecordMovement(t *testing.T) {
	t.Run("registra un ingreso y un egreso con su motivo", func(t *testing.T) {
		s := openSession(t, "100000")
		for _, kind := range []domain.CashMovementKind{domain.CashIn, domain.CashOut} {
			in := movement(s, kind, "20000", "  Pago del almuerzo  ")
			m, err := s.RecordMovement(in)
			require.NoError(t, err)
			require.Equal(t, in.ID, m.ID())
			require.Equal(t, s.ID(), m.SessionID())
			require.Equal(t, s.TenantID(), m.TenantID())
			require.Equal(t, kind, m.Kind())
			requireDec(t, "20000", m.Amount())
			require.Equal(t, "Pago del almuerzo", m.Reason())
			require.Equal(t, s.UserID(), m.UserID())
			require.Equal(t, in.At.UTC(), m.At())
		}
	})

	t.Run("rechaza un movimiento inválido", func(t *testing.T) {
		s := openSession(t, "0")
		tests := []struct {
			name   string
			change func(*domain.CashMovementInput)
			want   error
		}{
			{"sin id", func(in *domain.CashMovementInput) { in.ID = uuid.Nil }, domain.ErrInvalidCashMovementID},
			{"de tipo inválido", func(in *domain.CashMovementInput) { in.Kind = "transfer" }, domain.ErrInvalidCashMovementKind},
			{"con monto cero", func(in *domain.CashMovementInput) { in.Amount = dec("0") }, domain.ErrInvalidCashMovementAmount},
			{"con monto negativo", func(in *domain.CashMovementInput) { in.Amount = dec("-5000") }, domain.ErrInvalidCashMovementAmount},
			{"con monto de más de 2 decimales", func(in *domain.CashMovementInput) { in.Amount = dec("10.001") }, domain.ErrInvalidCashMovementAmount},
			{"sin motivo", func(in *domain.CashMovementInput) { in.Reason = "   " }, domain.ErrInvalidCashMovementReason},
			{"con motivo de más de 200 caracteres", func(in *domain.CashMovementInput) { in.Reason = strings.Repeat("a", 201) }, domain.ErrInvalidCashMovementReason},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				in := movement(s, domain.CashIn, "1000", "Base extra")
				tt.change(&in)
				_, err := s.RecordMovement(in)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})

	t.Run("solo quien abrió la caja registra movimientos", func(t *testing.T) {
		s := openSession(t, "0")
		in := movement(s, domain.CashOut, "1000", "Domicilio")
		in.UserID = newID()
		_, err := s.RecordMovement(in)
		require.ErrorIs(t, err, domain.ErrCashSessionNotOwner)
	})

	t.Run("no registra movimientos en una caja cerrada", func(t *testing.T) {
		s := openSession(t, "0")
		require.NoError(t, s.Close(closeInput(s, count(t), activity("0", "0", "0", "0"))))
		_, err := s.RecordMovement(movement(s, domain.CashIn, "1000", "Base extra"))
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
	})
}

func TestNewCashCount(t *testing.T) {
	t.Run("suma el conteo por denominación", func(t *testing.T) {
		c := count(t, "50000", 3, "2000", 4, "500", 7, "50", 1)
		requireDec(t, "161550", c.Total())
		require.Len(t, c.Counts(), 4)
	})

	t.Run("ordena las denominaciones de mayor a menor", func(t *testing.T) {
		c := count(t, "100", 2, "100000", 1, "5000", 0)
		got := c.Counts()
		requireDec(t, "100000", got[0].Denomination)
		requireDec(t, "5000", got[1].Denomination)
		requireDec(t, "100", got[2].Denomination)
	})

	t.Run("un conteo vacío suma cero", func(t *testing.T) {
		requireDec(t, "0", count(t).Total())
	})

	t.Run("acepta todas las denominaciones vigentes del peso colombiano", func(t *testing.T) {
		for _, d := range domain.Denominations() {
			_, err := domain.NewCashCount([]domain.DenominationCount{{Denomination: d, Quantity: 1}})
			require.NoError(t, err, d.String())
		}
		require.Len(t, domain.Denominations(), 11)
	})

	t.Run("rechaza un conteo inválido", func(t *testing.T) {
		tests := []struct {
			name   string
			counts []domain.DenominationCount
			want   error
		}{
			{"denominación que no existe", []domain.DenominationCount{{Denomination: dec("30000"), Quantity: 1}}, domain.ErrInvalidDenomination},
			{"denominación con centavos", []domain.DenominationCount{{Denomination: dec("50.5"), Quantity: 1}}, domain.ErrInvalidDenomination},
			{"denominación repetida", []domain.DenominationCount{{Denomination: dec("1000"), Quantity: 1}, {Denomination: dec("1000.00"), Quantity: 2}}, domain.ErrDuplicateDenomination},
			{"cantidad negativa", []domain.DenominationCount{{Denomination: dec("1000"), Quantity: -1}}, domain.ErrInvalidDenominationQuantity},
			{"cantidad desmedida", []domain.DenominationCount{{Denomination: dec("100000"), Quantity: 10_000_001}}, domain.ErrInvalidDenominationQuantity},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewCashCount(tt.counts)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}

func TestCashSessionClose(t *testing.T) {
	// Base 100000 + 250000 received − 20000 change + 10000 in − 40000 out = 300000.
	act := activity("250000", "20000", "10000", "40000")

	tests := []struct {
		name       string
		count      []any
		counted    string
		difference string
		result     domain.CashResult
	}{
		{name: "cuadra en cero cuando el conteo es igual al esperado", count: []any{"100000", 3}, counted: "300000", difference: "0", result: domain.CashExact},
		{name: "calcula el faltante cuando se cuenta menos que el esperado", count: []any{"100000", 2, "50000", 1, "20000", 2, "5000", 1, "200", 1}, counted: "295200", difference: "-4800", result: domain.CashShortage},
		{name: "calcula el sobrante cuando se cuenta más que el esperado", count: []any{"100000", 3, "500", 1}, counted: "300500", difference: "500", result: domain.CashOverage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := openSession(t, "100000")
			in := closeInput(s, count(t, tt.count...), act)
			in.Notes = "  Turno de la mañana  "

			require.NoError(t, s.Close(in))

			require.Equal(t, domain.CashSessionClosed, s.Status())
			c := s.Closing()
			require.NotNil(t, c)
			requireDec(t, "300000", c.Expected)
			requireDec(t, tt.counted, c.Counted)
			requireDec(t, tt.difference, c.Difference)
			require.Equal(t, tt.result, c.Result())
			require.Equal(t, s.UserID(), c.ClosedBy)
			require.Equal(t, in.At.UTC(), c.ClosedAt)
			require.Equal(t, "Turno de la mañana", c.Notes)
			requireDec(t, "250000", c.Activity.CashReceived)
		})
	}

	t.Run("el cierre queda inmutable: no se puede cerrar dos veces", func(t *testing.T) {
		s := openSession(t, "0")
		require.NoError(t, s.Close(closeInput(s, count(t), activity("0", "0", "0", "0"))))
		first := *s.Closing()

		err := s.Close(closeInput(s, count(t, "1000", 1), activity("5000", "0", "0", "0")))
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
		require.Equal(t, first, *s.Closing())
	})

	t.Run("solo quien abrió la caja la cierra", func(t *testing.T) {
		s := openSession(t, "0")
		in := closeInput(s, count(t), activity("0", "0", "0", "0"))
		in.UserID = newID()
		require.ErrorIs(t, s.Close(in), domain.ErrCashSessionNotOwner)
		require.Equal(t, domain.CashSessionOpen, s.Status())
	})

	t.Run("rechaza notas de más de 500 caracteres", func(t *testing.T) {
		s := openSession(t, "0")
		in := closeInput(s, count(t), activity("0", "0", "0", "0"))
		in.Notes = strings.Repeat("a", 501)
		require.ErrorIs(t, s.Close(in), domain.ErrInvalidCashNotes)
		require.Nil(t, s.Closing())
	})

	t.Run("rechaza una actividad con montos negativos", func(t *testing.T) {
		s := openSession(t, "0")
		require.ErrorIs(t, s.Close(closeInput(s, count(t), activity("-1", "0", "0", "0"))), domain.ErrInvalidCashActivity)
		require.ErrorIs(t, s.Close(closeInput(s, count(t), activity("0", "-1", "0", "0"))), domain.ErrInvalidCashActivity)
		require.Equal(t, domain.CashSessionOpen, s.Status())
	})
}

func TestRehydrateCashSession(t *testing.T) {
	closed := func(t *testing.T) *domain.CashSession {
		t.Helper()
		s := openSession(t, "100000")
		a := activity("250000", "20000", "0", "0")
		a.Sales = 4
		a.ByMethod = []domain.MethodTotal{
			{Method: domain.PaymentCash, Amount: dec("230000")},
			{Method: domain.PaymentCard, Amount: dec("80000")},
		}
		require.NoError(t, s.Close(closeInput(s, count(t, "100000", 3), a)))
		return s
	}

	t.Run("rehidrata una caja abierta y una cerrada", func(t *testing.T) {
		for _, s := range []*domain.CashSession{openSession(t, "50000"), closed(t)} {
			got, err := domain.RehydrateCashSession(s.Snapshot())
			require.NoError(t, err)
			require.Equal(t, s.Snapshot(), got.Snapshot())
		}
	})

	t.Run("rechaza montos guardados que no cuadran", func(t *testing.T) {
		tests := []struct {
			name   string
			change func(*domain.CashSessionSnapshot)
		}{
			{"esperado distinto", func(s *domain.CashSessionSnapshot) { s.Closing.Expected = dec("1") }},
			{"contado distinto del conteo", func(s *domain.CashSessionSnapshot) { s.Closing.Counted = dec("1") }},
			{"diferencia distinta", func(s *domain.CashSessionSnapshot) { s.Closing.Difference = dec("1") }},
			{"cerrada sin cierre", func(s *domain.CashSessionSnapshot) { s.Closing = nil }},
			{"estado desconocido", func(s *domain.CashSessionSnapshot) { s.Status = "lost" }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				snap := closed(t).Snapshot()
				tt.change(&snap)
				_, err := domain.RehydrateCashSession(snap)
				require.ErrorIs(t, err, domain.ErrInconsistentCashSession)
			})
		}
	})
}

func TestSaleCashSession(t *testing.T) {
	t.Run("la venta guarda la caja donde se cobró", func(t *testing.T) {
		session := newID()
		s, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CashSessionID: &session, CreatedAt: testNow,
		})
		require.NoError(t, err)
		require.Equal(t, &session, s.CashSessionID())
	})

	t.Run("rechaza una caja vacía", func(t *testing.T) {
		nilID := uuid.Nil
		_, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CashSessionID: &nilID, CreatedAt: testNow,
		})
		require.ErrorIs(t, err, domain.ErrInvalidCashSessionID)
	})
}
