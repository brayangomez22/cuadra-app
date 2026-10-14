package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func (s *stubUseCases) OpenCashSession(_ context.Context, a app.Actor, in app.OpenCashSessionInput) (*domain.CashSession, error) {
	s.actor, s.openCash = a, in
	return s.session, s.err
}

func (s *stubUseCases) RecordCashMovement(_ context.Context, a app.Actor, id uuid.UUID, in app.CashMovementRequest) (domain.CashMovement, error) {
	s.actor, s.id, s.movement = a, id, in
	return s.recorded, s.err
}

func (s *stubUseCases) CloseCashSession(_ context.Context, a app.Actor, id uuid.UUID, in app.CloseCashSessionInput) (app.CashSessionView, error) {
	s.actor, s.id, s.closing = a, id, in
	return s.view, s.err
}

func (s *stubUseCases) GetCashSession(_ context.Context, a app.Actor, id uuid.UUID) (app.CashSessionView, error) {
	s.actor, s.id = a, id
	return s.view, s.err
}

func (s *stubUseCases) GetCurrentCashSession(_ context.Context, a app.Actor) (app.CurrentCashSession, error) {
	s.actor = a
	return s.current, s.err
}

// anOpenSession is the owner's session with a float of 100000 and a cash out
// of 12000.
func anOpenSession(t *testing.T) app.CashSessionView {
	t.Helper()
	s, err := domain.OpenCashSession(domain.NewCashSessionParams{
		ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, OpeningFloat: dec("100000"), OpenedAt: now,
	})
	require.NoError(t, err)
	m, err := s.RecordMovement(domain.CashMovementInput{
		ID: newID(), Kind: domain.CashOut, Amount: dec("12000"), Reason: "Almuerzo", UserID: owner.UserID, At: now,
	})
	require.NoError(t, err)
	return app.CashSessionView{Session: s, Movements: []domain.CashMovement{m}}
}

// aClosedSession is anOpenSession closed with 150000 counted: expected
// 100000 + 80000 − 12000 − 20000 = 148000, overage 2000.
func aClosedSession(t *testing.T) app.CashSessionView {
	t.Helper()
	v := anOpenSession(t)
	count, err := domain.NewCashCount([]domain.DenominationCount{
		{Denomination: dec("100000"), Quantity: 1}, {Denomination: dec("50000"), Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, v.Session.Close(domain.CloseCashInput{
		UserID: owner.UserID, Count: count, Notes: "Sin novedad", At: now,
		Activity: domain.CashActivity{
			CashReceived: dec("80000"), ChangeGiven: dec("20000"), CashIn: dec("0"), CashOut: dec("12000"), Sales: 3,
			ByMethod: []domain.MethodTotal{{Method: domain.PaymentCash, Amount: dec("60000")}, {Method: domain.PaymentCard, Amount: dec("45000")}},
		},
	}))
	return v
}

type sessionBody struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	OpeningFloat string    `json:"opening_float"`
	Movements    []struct {
		Kind, Amount, Reason string
	} `json:"movements"`
	Closing *struct {
		SalesCount   int    `json:"sales_count"`
		CashReceived string `json:"cash_received"`
		ChangeGiven  string `json:"change_given"`
		CashOut      string `json:"cash_out"`
		Expected     string `json:"expected"`
		Counted      string `json:"counted"`
		Difference   string `json:"difference"`
		Result       string `json:"result"`
		Notes        string `json:"notes"`
		Counts       []struct {
			Denomination string `json:"denomination"`
			Quantity     int64  `json:"quantity"`
		} `json:"counts"`
		Payments []struct{ Method, Amount string } `json:"payments"`
	} `json:"closing"`
}

func decodeSession(t *testing.T, raw []byte) sessionBody {
	t.Helper()
	var body sessionBody
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func TestOpenCashSession(t *testing.T) {
	t.Run("abre la caja del usuario autenticado con su base", func(t *testing.T) {
		v := anOpenSession(t)
		uc := &stubUseCases{session: v.Session}
		location := newID()

		p := cashier()
		rec := serve(t, uc, &p, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+location.String()+`","opening_float":"100000"}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, owner.UserID, uc.actor.UserID)
		require.Equal(t, location, uc.openCash.LocationID)
		require.True(t, dec("100000").Equal(uc.openCash.OpeningFloat))

		body := decodeSession(t, rec.Body.Bytes())
		require.Equal(t, "open", body.Status)
		require.Equal(t, "100000.00", body.OpeningFloat)
		require.Empty(t, body.Movements)
		require.Nil(t, body.Closing)
	})

	t.Run("una segunda caja responde 409", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCashSessionAlreadyOpen}
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+newID().String()+`","opening_float":"0"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Contains(t, rec.Body.String(), `"cash_session_already_open"`)
	})
}

func TestGetCurrentCashSession(t *testing.T) {
	t.Run("sin caja abierta dice si la empresa la exige", func(t *testing.T) {
		uc := &stubUseCases{current: app.CurrentCashSession{Required: true}}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/cash-sessions/current", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"required":true}`, rec.Body.String())
	})

	t.Run("no muestra el esperado de una caja abierta (arqueo ciego)", func(t *testing.T) {
		v := anOpenSession(t)
		uc := &stubUseCases{current: app.CurrentCashSession{Required: true, Session: &v}}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/cash-sessions/current", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotContains(t, rec.Body.String(), "expected")
		require.NotContains(t, rec.Body.String(), "closing")

		var body struct{ Session sessionBody }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, v.Session.ID(), body.Session.ID)
		require.Len(t, body.Session.Movements, 1)
		require.Equal(t, "out", body.Session.Movements[0].Kind)
		require.Equal(t, "12000.00", body.Session.Movements[0].Amount)
		require.Equal(t, "Almuerzo", body.Session.Movements[0].Reason)
	})
}

func TestRecordCashMovement(t *testing.T) {
	t.Run("registra el movimiento en la caja indicada", func(t *testing.T) {
		v := anOpenSession(t)
		uc := &stubUseCases{recorded: v.Movements[0]}
		id := v.Session.ID()

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/cash-sessions/"+id.String()+"/movements",
			`{"kind":"out","amount":"12000","reason":"Almuerzo"}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
		require.Equal(t, app.CashMovementRequest{Kind: "out", Amount: dec("12000"), Reason: "Almuerzo"}, uc.movement)
		require.Contains(t, rec.Body.String(), `"amount":"12000.00"`)
	})

	t.Run("mapea los errores de la caja a su status y código", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrCashSessionClosed, http.StatusConflict, "cash_session_closed"},
			{domain.ErrCashSessionNotOwner, http.StatusForbidden, "cash_session_not_owner"},
			{domain.ErrCashSessionNotFound, http.StatusNotFound, "cash_session_not_found"},
			{domain.ErrInvalidCashMovementAmount, http.StatusUnprocessableEntity, "invalid_cash_movement_amount"},
			{domain.ErrInvalidCashMovementReason, http.StatusUnprocessableEntity, "invalid_cash_movement_reason"},
		}
		for _, tt := range tests {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPost, "/api/v1/cash-sessions/"+newID().String()+"/movements",
				`{"kind":"in","amount":"1000","reason":"Base"}`)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"`+tt.code+`"`)
		}
	})
}

func TestCloseCashSession(t *testing.T) {
	t.Run("cierra con el conteo y responde el reporte de cierre", func(t *testing.T) {
		v := aClosedSession(t)
		uc := &stubUseCases{view: v}
		id := v.Session.ID()

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/cash-sessions/"+id.String()+"/close",
			`{"counts":[{"denomination":"100000","quantity":1},{"denomination":"50000","quantity":1}],"notes":"Sin novedad"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
		require.Len(t, uc.closing.Counts, 2)
		require.True(t, dec("50000").Equal(uc.closing.Counts[1].Denomination))
		require.Equal(t, int64(1), uc.closing.Counts[1].Quantity)
		require.Equal(t, "Sin novedad", uc.closing.Notes)

		body := decodeSession(t, rec.Body.Bytes())
		require.Equal(t, "closed", body.Status)
		c := body.Closing
		require.NotNil(t, c)
		require.Equal(t, 3, c.SalesCount)
		require.Equal(t, "80000.00", c.CashReceived)
		require.Equal(t, "20000.00", c.ChangeGiven)
		require.Equal(t, "12000.00", c.CashOut)
		require.Equal(t, "148000.00", c.Expected)
		require.Equal(t, "150000.00", c.Counted)
		require.Equal(t, "2000.00", c.Difference)
		require.Equal(t, "overage", c.Result)
		require.Equal(t, "Sin novedad", c.Notes)
		require.Len(t, c.Counts, 2)
		require.Equal(t, "100000.00", c.Counts[0].Denomination)
		require.Len(t, c.Payments, 2)
		require.Equal(t, "cash", c.Payments[0].Method)
		require.Equal(t, "60000.00", c.Payments[0].Amount)
	})

	t.Run("una diferencia negativa viaja con signo", func(t *testing.T) {
		v := anOpenSession(t)
		require.NoError(t, v.Session.Close(domain.CloseCashInput{
			UserID: owner.UserID, At: now,
			Activity: domain.CashActivity{CashReceived: dec("0"), ChangeGiven: dec("0"), CashIn: dec("0"), CashOut: dec("12000")},
		}))
		rec := serve(t, &stubUseCases{view: v}, &owner, http.MethodPost, "/api/v1/cash-sessions/"+v.Session.ID().String()+"/close", `{"counts":[]}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decodeSession(t, rec.Body.Bytes())
		require.Equal(t, "-88000.00", body.Closing.Difference)
		require.Equal(t, "shortage", body.Closing.Result)
	})

	t.Run("mapea los errores del conteo", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrInvalidDenomination, http.StatusUnprocessableEntity, "invalid_denomination"},
			{domain.ErrDuplicateDenomination, http.StatusUnprocessableEntity, "duplicate_denomination"},
			{domain.ErrInvalidDenominationQuantity, http.StatusUnprocessableEntity, "invalid_denomination_quantity"},
			{domain.ErrInvalidCashNotes, http.StatusUnprocessableEntity, "invalid_cash_notes"},
			{domain.ErrCashSessionClosed, http.StatusConflict, "cash_session_closed"},
		}
		for _, tt := range tests {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPost, "/api/v1/cash-sessions/"+newID().String()+"/close", `{"counts":[]}`)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"`+tt.code+`"`)
		}
	})
}

func TestGetCashSession(t *testing.T) {
	t.Run("responde la caja con su cierre", func(t *testing.T) {
		v := aClosedSession(t)
		uc := &stubUseCases{view: v}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/cash-sessions/"+v.Session.ID().String(), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, v.Session.ID(), uc.id)
		require.Equal(t, "148000.00", decodeSession(t, rec.Body.Bytes()).Closing.Expected)
	})
}

func TestConfirmSaleCashErrors(t *testing.T) {
	t.Run("mapea los errores de la caja al confirmar una venta", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrCashSessionRequired, http.StatusConflict, "cash_session_required"},
			{domain.ErrCashSessionClosed, http.StatusConflict, "cash_session_closed"},
			{domain.ErrCashSessionLocationMismatch, http.StatusUnprocessableEntity, "cash_session_location_mismatch"},
		}
		body := `{"location_id":"` + newID().String() + `","lines":[{"product_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`
		for _, tt := range tests {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPost, "/api/v1/sales", body)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"`+tt.code+`"`)
		}
	})

	t.Run("la venta responde la caja donde se cobró", func(t *testing.T) {
		session := newID()
		s, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, CashSessionID: &session, CreatedAt: now,
		})
		require.NoError(t, err)
		require.NoError(t, s.AddLine(domain.LineInput{
			ProductID: newID(), ProductName: "Puntilla", Quantity: dec("1"), UnitPrice: dec("100"), TaxRate: dec("0"), Discount: dec("0"),
		}, domain.NoDiscountLimit))
		cash, err := domain.NewPayment(domain.PaymentCash, dec("100"))
		require.NoError(t, err)
		require.NoError(t, s.Confirm([]domain.Payment{cash}, now))

		rec := serve(t, &stubUseCases{sale: s}, &owner, http.MethodGet, "/api/v1/sales/"+s.ID().String(), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"cash_session_id":"`+session.String()+`"`)
	})
}
