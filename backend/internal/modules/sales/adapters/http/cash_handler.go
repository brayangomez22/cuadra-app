package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http/salesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// OpenCashSession opens the caller's cash session.
func (h *Handler) OpenCashSession(ctx context.Context, req salesapi.OpenCashSessionRequestObject) (salesapi.OpenCashSessionResponseObject, error) {
	float, err := parseDecimal(req.Body.OpeningFloat, domain.ErrInvalidOpeningFloat)
	if err != nil {
		return nil, err
	}
	session, err := h.uc.OpenCashSession(ctx, h.actor(ctx), app.OpenCashSessionInput{LocationID: req.Body.LocationId, OpeningFloat: float})
	if err != nil {
		return nil, err
	}
	return salesapi.OpenCashSession201JSONResponse(sessionBody(app.CashSessionView{Session: session})), nil
}

// GetCurrentCashSession returns the caller's open session, if any, and
// whether the tenant requires one to sell.
func (h *Handler) GetCurrentCashSession(ctx context.Context, _ salesapi.GetCurrentCashSessionRequestObject) (salesapi.GetCurrentCashSessionResponseObject, error) {
	current, err := h.uc.GetCurrentCashSession(ctx, h.actor(ctx))
	if err != nil {
		return nil, err
	}
	body := salesapi.GetCurrentCashSession200JSONResponse{Required: current.Required}
	if current.Session != nil {
		s := sessionBody(*current.Session)
		body.Session = &s
	}
	return body, nil
}

// GetCashSession returns a session with its movements and closing.
func (h *Handler) GetCashSession(ctx context.Context, req salesapi.GetCashSessionRequestObject) (salesapi.GetCashSessionResponseObject, error) {
	view, err := h.uc.GetCashSession(ctx, h.actor(ctx), req.CashSessionId)
	if err != nil {
		return nil, err
	}
	return salesapi.GetCashSession200JSONResponse(sessionBody(view)), nil
}

// RecordCashMovement records a cash in or out.
func (h *Handler) RecordCashMovement(ctx context.Context, req salesapi.RecordCashMovementRequestObject) (salesapi.RecordCashMovementResponseObject, error) {
	amount, err := parseDecimal(req.Body.Amount, domain.ErrInvalidCashMovementAmount)
	if err != nil {
		return nil, err
	}
	m, err := h.uc.RecordCashMovement(ctx, h.actor(ctx), req.CashSessionId, app.CashMovementRequest{
		Kind: string(req.Body.Kind), Amount: amount, Reason: req.Body.Reason,
	})
	if err != nil {
		return nil, err
	}
	return salesapi.RecordCashMovement201JSONResponse(movementBody(m)), nil
}

// CloseCashSession closes the caller's session with the count of the drawer.
func (h *Handler) CloseCashSession(ctx context.Context, req salesapi.CloseCashSessionRequestObject) (salesapi.CloseCashSessionResponseObject, error) {
	in := app.CloseCashSessionInput{}
	if req.Body.Notes != nil {
		in.Notes = *req.Body.Notes
	}
	for _, c := range req.Body.Counts {
		d, err := parseDecimal(c.Denomination, domain.ErrInvalidDenomination)
		if err != nil {
			return nil, err
		}
		in.Counts = append(in.Counts, domain.DenominationCount{Denomination: d, Quantity: c.Quantity})
	}
	view, err := h.uc.CloseCashSession(ctx, h.actor(ctx), req.CashSessionId, in)
	if err != nil {
		return nil, err
	}
	return salesapi.CloseCashSession200JSONResponse(sessionBody(view)), nil
}

// sessionBody writes a session. An open session has no closing: the expected
// cash is only shown once the drawer was counted (arqueo ciego).
func sessionBody(v app.CashSessionView) salesapi.CashSession {
	s := v.Session
	body := salesapi.CashSession{
		Id: s.ID(), LocationId: s.LocationID(), UserId: s.UserID(), Status: salesapi.CashSessionStatus(s.Status()),
		OpeningFloat: formatMoney(s.OpeningFloat()), OpenedAt: s.OpenedAt(),
		Movements: make([]salesapi.CashMovement, 0, len(v.Movements)),
	}
	for _, m := range v.Movements {
		body.Movements = append(body.Movements, movementBody(m))
	}
	if c := s.Closing(); c != nil {
		a := c.Activity
		closing := salesapi.CashClosing{
			ClosedAt: c.ClosedAt, ClosedBy: c.ClosedBy, SalesCount: a.Sales,
			CashReceived: formatMoney(a.CashReceived), ChangeGiven: formatMoney(a.ChangeGiven),
			CashIn: formatMoney(a.CashIn), CashOut: formatMoney(a.CashOut),
			Expected: formatMoney(c.Expected), Counted: formatMoney(c.Counted), Difference: formatMoney(c.Difference),
			Result:   salesapi.CashClosingResult(c.Result()),
			Counts:   make([]salesapi.DenominationCount, 0, len(c.Count.Counts())),
			Payments: make([]salesapi.CashMethodTotal, 0, len(a.ByMethod)),
		}
		if c.Notes != "" {
			notes := c.Notes
			closing.Notes = &notes
		}
		for _, d := range c.Count.Counts() {
			closing.Counts = append(closing.Counts, salesapi.DenominationCount{Denomination: formatMoney(d.Denomination), Quantity: d.Quantity})
		}
		for _, p := range a.ByMethod {
			closing.Payments = append(closing.Payments, salesapi.CashMethodTotal{Method: salesapi.PaymentMethod(p.Method), Amount: formatMoney(p.Amount)})
		}
		body.Closing = &closing
	}
	return body
}

func movementBody(m domain.CashMovement) salesapi.CashMovement {
	return salesapi.CashMovement{
		Id: m.ID(), Kind: salesapi.CashMovementKind(m.Kind()), Amount: formatMoney(m.Amount()),
		Reason: m.Reason(), UserId: m.UserID(), CreatedAt: m.At(),
	}
}
