package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// openCash opens a cash session for the actor at location with float.
func (e *env) openCash(t *testing.T, location uuid.UUID, float string) *domain.CashSession {
	t.Helper()
	s, err := e.svc.OpenCashSession(t.Context(), e.actor, app.OpenCashSessionInput{LocationID: location, OpeningFloat: dec(float)})
	require.NoError(t, err)
	return s
}

// sellFor confirms a sale of one product at location paid with payments.
func (e *env) sellFor(t *testing.T, location uuid.UUID, price string, payments ...app.PaymentRequest) (*domain.Sale, error) {
	t.Helper()
	id := e.addProduct("Martillo", price, "0", true)
	return e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
		LocationID: location,
		Lines:      []app.LineRequest{lineReq(id, "1", "0")},
		Payments:   payments,
	})
}

func counts(pairs ...any) []domain.DenominationCount {
	var out []domain.DenominationCount
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, domain.DenominationCount{Denomination: dec(pairs[i].(string)), Quantity: int64(pairs[i+1].(int))})
	}
	return out
}

func TestOpenCashSession(t *testing.T) {
	t.Run("abre la caja del usuario en la sede con su base", func(t *testing.T) {
		e := newEnv(t)
		location := newID()

		s := e.openCash(t, location, "150000")

		require.Equal(t, domain.CashSessionOpen, s.Status())
		require.Equal(t, e.actor.TenantID, s.TenantID())
		require.Equal(t, e.actor.UserID, s.UserID())
		require.Equal(t, location, s.LocationID())
		require.Equal(t, 7, int(s.ID().Version()))
		requireDec(t, "150000", s.OpeningFloat())
		require.Equal(t, testNow.UTC(), s.OpenedAt())

		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.cash_sessions_opened"))
		span, attrs := e.span(t, "sales.OpenCashSession")
		require.Equal(t, codes.Unset, span.Status().Code)
		require.Equal(t, s.ID().String(), attrs["cash_session.id"].AsString())
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("rechaza abrir una segunda caja si el usuario ya tiene una abierta", func(t *testing.T) {
		e := newEnv(t)
		e.openCash(t, newID(), "0")

		_, err := e.svc.OpenCashSession(t.Context(), e.actor, app.OpenCashSessionInput{LocationID: newID(), OpeningFloat: dec("0")})
		require.ErrorIs(t, err, domain.ErrCashSessionAlreadyOpen)

		spans := e.spans.Ended()
		last := spans[len(spans)-1]
		require.Equal(t, "sales.OpenCashSession", last.Name())
		require.Equal(t, codes.Error, last.Status().Code)
	})

	t.Run("otro usuario de la empresa abre su propia caja", func(t *testing.T) {
		e := newEnv(t)
		e.openCash(t, newID(), "0")
		e.actor.UserID = newID()
		e.openCash(t, newID(), "0")
	})
}

func TestConfirmSaleWithCashSession(t *testing.T) {
	t.Run("la venta queda asociada a la caja abierta del usuario", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()
		location := newID()
		session := e.openCash(t, location, "50000")

		sale, err := e.sellFor(t, location, "10000", cash("20000"))
		require.NoError(t, err)
		require.Equal(t, session.ID(), *sale.CashSessionID())
		require.Equal(t, []uuid.UUID{session.ID()}, e.store.ensured, "verifica la caja dentro de la transacción")
	})

	t.Run("asocia también las ventas con otros medios de pago", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()
		location := newID()
		session := e.openCash(t, location, "0")

		sale, err := e.sellFor(t, location, "10000", app.PaymentRequest{Method: "card", Amount: dec("10000")})
		require.NoError(t, err)
		require.Equal(t, session.ID(), *sale.CashSessionID())
	})

	t.Run("rechaza la venta sin caja abierta cuando la empresa lo exige", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()

		_, err := e.sellFor(t, newID(), "10000", cash("10000"))
		require.ErrorIs(t, err, domain.ErrCashSessionRequired)
		require.Zero(t, e.storedSales())
		require.Empty(t, e.store.reserved)
		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.cash_session_required_rejected"))
	})

	t.Run("permite vender sin caja cuando la empresa no lo exige", func(t *testing.T) {
		e := newEnv(t)

		sale, err := e.sellFor(t, newID(), "10000", cash("10000"))
		require.NoError(t, err)
		require.Nil(t, sale.CashSessionID())
		require.Empty(t, e.store.ensured)
	})

	t.Run("aunque no lo exija, asocia la venta si el usuario tiene caja abierta", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		session := e.openCash(t, location, "0")

		sale, err := e.sellFor(t, location, "10000", cash("10000"))
		require.NoError(t, err)
		require.Equal(t, session.ID(), *sale.CashSessionID())
	})

	t.Run("rechaza la venta en una sede distinta a la de la caja", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()
		e.openCash(t, newID(), "0")

		_, err := e.sellFor(t, newID(), "10000", cash("10000"))
		require.ErrorIs(t, err, domain.ErrCashSessionLocationMismatch)
		require.Zero(t, e.storedSales())
	})

	t.Run("no se pueden registrar ventas en una caja cerrada", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()
		location := newID()
		session := e.openCash(t, location, "0")
		// The session closes between the lookup and the save.
		e.store.onEnsureOpen = func() {
			_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{})
			require.NoError(t, err)
		}

		_, err := e.sellFor(t, location, "10000", cash("10000"))
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
		require.Zero(t, e.storedSales())
		require.Empty(t, e.store.reserved)
	})
}

func TestRecordCashMovement(t *testing.T) {
	t.Run("registra un egreso con motivo en la caja del usuario", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "100000")

		m, err := e.svc.RecordCashMovement(t.Context(), e.actor, session.ID(), app.CashMovementRequest{
			Kind: "out", Amount: dec("15000"), Reason: "Pago del domicilio",
		})
		require.NoError(t, err)
		require.Equal(t, domain.CashOut, m.Kind())
		requireDec(t, "15000", m.Amount())
		require.Equal(t, "Pago del domicilio", m.Reason())
		require.Equal(t, session.ID(), m.SessionID())
		require.Equal(t, e.actor.UserID, m.UserID())
		require.Equal(t, 7, int(m.ID().Version()))

		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.cash_movements"))
		_, attrs := e.span(t, "sales.RecordCashMovement")
		require.Equal(t, session.ID().String(), attrs["cash_session.id"].AsString())
	})

	t.Run("rechaza un tipo de movimiento inválido", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		_, err := e.svc.RecordCashMovement(t.Context(), e.actor, session.ID(), app.CashMovementRequest{
			Kind: "loan", Amount: dec("1000"), Reason: "x",
		})
		require.ErrorIs(t, err, domain.ErrInvalidCashMovementKind)
	})

	t.Run("no permite registrar movimientos en la caja de otro usuario", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		other := e.actor
		other.UserID = newID()

		_, err := e.svc.RecordCashMovement(t.Context(), other, session.ID(), app.CashMovementRequest{
			Kind: "in", Amount: dec("1000"), Reason: "Base extra",
		})
		require.ErrorIs(t, err, domain.ErrCashSessionNotOwner)
		require.Empty(t, e.store.movements)
	})

	t.Run("una caja de otra empresa no existe", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		other := app.Actor{TenantID: newID(), UserID: e.actor.UserID}

		_, err := e.svc.RecordCashMovement(t.Context(), other, session.ID(), app.CashMovementRequest{
			Kind: "in", Amount: dec("1000"), Reason: "Base extra",
		})
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
	})
}

func TestCloseCashSession(t *testing.T) {
	t.Run("cierra con el esperado calculado de las ventas en efectivo, el cambio y los movimientos", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		session := e.openCash(t, location, "100000")

		// Cash 30000 for 25000: change 5000. Card 40000. Mixed: 10000 card + 20000 cash for 28000: change 2000.
		_, err := e.sellFor(t, location, "25000", cash("30000"))
		require.NoError(t, err)
		_, err = e.sellFor(t, location, "40000", app.PaymentRequest{Method: "card", Amount: dec("40000")})
		require.NoError(t, err)
		_, err = e.sellFor(t, location, "28000", app.PaymentRequest{Method: "card", Amount: dec("10000")}, cash("20000"))
		require.NoError(t, err)
		_, err = e.svc.RecordCashMovement(t.Context(), e.actor, session.ID(), app.CashMovementRequest{Kind: "in", Amount: dec("20000"), Reason: "Más sencillo"})
		require.NoError(t, err)
		_, err = e.svc.RecordCashMovement(t.Context(), e.actor, session.ID(), app.CashMovementRequest{Kind: "out", Amount: dec("12000"), Reason: "Almuerzo"})
		require.NoError(t, err)

		// 100000 + 50000 − 7000 + 20000 − 12000 = 151000; counted 150000.
		view, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{
			Counts: counts("100000", 1, "50000", 1), Notes: "Faltó un billete",
		})
		require.NoError(t, err)

		c := view.Session.Closing()
		require.NotNil(t, c)
		requireDec(t, "50000", c.Activity.CashReceived)
		requireDec(t, "7000", c.Activity.ChangeGiven)
		requireDec(t, "20000", c.Activity.CashIn)
		requireDec(t, "12000", c.Activity.CashOut)
		require.Equal(t, 3, c.Activity.Sales)
		requireDec(t, "151000", c.Expected)
		requireDec(t, "150000", c.Counted)
		requireDec(t, "-1000", c.Difference)
		require.Equal(t, domain.CashShortage, c.Result())
		require.Equal(t, "Faltó un billete", c.Notes)
		require.Len(t, view.Movements, 2)

		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.cash_sessions_closed"))
		n, sum := e.histogram(t, "cuadra.sales.cash_difference")
		require.Equal(t, uint64(1), n)
		require.InDelta(t, 1000, sum, 0.001, "registra el valor absoluto de la diferencia")
		_, attrs := e.span(t, "sales.CloseCashSession")
		require.Equal(t, session.ID().String(), attrs["cash_session.id"].AsString())
	})

	t.Run("las ventas anuladas antes del cierre no cuentan en el esperado", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		session := e.openCash(t, location, "0")
		sale, err := e.sellFor(t, location, "10000", cash("10000"))
		require.NoError(t, err)
		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Desistió")
		require.NoError(t, err)

		view, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{})
		require.NoError(t, err)
		requireDec(t, "0", view.Session.Closing().Expected)
		require.Equal(t, domain.CashExact, view.Session.Closing().Result())
	})

	t.Run("el cierre queda inmutable: cerrar de nuevo falla", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{})
		require.NoError(t, err)

		_, err = e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{Counts: counts("1000", 1)})
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
	})

	t.Run("no permite cerrar la caja de otro usuario", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		other := e.actor
		other.UserID = newID()

		_, err := e.svc.CloseCashSession(t.Context(), other, session.ID(), app.CloseCashSessionInput{})
		require.ErrorIs(t, err, domain.ErrCashSessionNotOwner)
	})

	t.Run("rechaza un conteo inválido", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{Counts: counts("30000", 1)})
		require.ErrorIs(t, err, domain.ErrInvalidDenomination)
	})

	t.Run("después del cierre el usuario puede abrir otra caja", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")
		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{})
		require.NoError(t, err)
		e.openCash(t, newID(), "0")
	})
}

func TestGetCashSessions(t *testing.T) {
	t.Run("la caja actual dice si la empresa la exige y trae sus movimientos", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()

		current, err := e.svc.GetCurrentCashSession(t.Context(), e.actor)
		require.NoError(t, err)
		require.True(t, current.Required)
		require.Nil(t, current.Session)

		session := e.openCash(t, newID(), "0")
		_, err = e.svc.RecordCashMovement(t.Context(), e.actor, session.ID(), app.CashMovementRequest{Kind: "in", Amount: dec("1000"), Reason: "Base"})
		require.NoError(t, err)

		current, err = e.svc.GetCurrentCashSession(t.Context(), e.actor)
		require.NoError(t, err)
		require.NotNil(t, current.Session)
		require.Equal(t, session.ID(), current.Session.Session.ID())
		require.Len(t, current.Session.Movements, 1)
	})

	t.Run("sin exigirla, la caja actual no es obligatoria", func(t *testing.T) {
		e := newEnv(t)
		current, err := e.svc.GetCurrentCashSession(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, current.Required)
	})

	t.Run("devuelve una caja con sus movimientos; la de otra empresa no existe", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "0")

		view, err := e.svc.GetCashSession(t.Context(), e.actor, session.ID())
		require.NoError(t, err)
		require.Equal(t, session.ID(), view.Session.ID())

		_, err = e.svc.GetCashSession(t.Context(), app.Actor{TenantID: newID(), UserID: newID()}, session.ID())
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
	})
}
