package app_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// fakeCash keeps cash sessions in memory. Its activity is computed from the
// stored sales as the SQL does: only confirmed sales of the session count.
type fakeCash struct{ *store }

func cloneSession(s *domain.CashSession) *domain.CashSession {
	c, err := domain.RehydrateCashSession(s.Snapshot())
	if err != nil {
		panic(err)
	}
	return c
}

func (f fakeCash) Open(_ context.Context, s *domain.CashSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.sessions {
		if other.TenantID() == s.TenantID() && other.UserID() == s.UserID() && other.Status() == domain.CashSessionOpen {
			return domain.ErrCashSessionAlreadyOpen
		}
	}
	f.sessions[s.ID()] = cloneSession(s)
	return nil
}

func (f fakeCash) get(tenantID, id uuid.UUID) (*domain.CashSession, error) {
	s, ok := f.sessions[id]
	if !ok || s.TenantID() != tenantID {
		return nil, domain.ErrCashSessionNotFound
	}
	return cloneSession(s), nil
}

func (f fakeCash) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.get(tenantID, id)
}

func (f fakeCash) Current(_ context.Context, tenantID, userID uuid.UUID) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if s.TenantID() == tenantID && s.UserID() == userID && s.Status() == domain.CashSessionOpen {
			return cloneSession(s), nil
		}
	}
	return nil, domain.ErrCashSessionNotFound
}

func (f fakeCash) Movements(_ context.Context, tenantID, id uuid.UUID) ([]domain.CashMovement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.CashMovement
	for _, m := range f.movements {
		if m.TenantID() == tenantID && m.SessionID() == id {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f fakeCash) AddMovement(_ context.Context, tenantID, id uuid.UUID, record func(*domain.CashSession) (domain.CashMovement, error)) (domain.CashMovement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.get(tenantID, id)
	if err != nil {
		return domain.CashMovement{}, err
	}
	m, err := record(s)
	if err != nil {
		return domain.CashMovement{}, err
	}
	f.movements = append(f.movements, m)
	return m, nil
}

func (f fakeCash) EnsureOpen(_ context.Context, tenantID, id uuid.UUID) error {
	if f.onEnsureOpen != nil {
		f.onEnsureOpen()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensured = append(f.ensured, id)
	s, err := f.get(tenantID, id)
	if err != nil {
		return err
	}
	if s.Status() != domain.CashSessionOpen {
		return domain.ErrCashSessionClosed
	}
	return nil
}

func (f fakeCash) Close(_ context.Context, tenantID, id uuid.UUID, closeFn func(*domain.CashSession, domain.CashActivity) error) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.get(tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := closeFn(s, f.activity(id)); err != nil {
		return nil, err
	}
	f.sessions[id] = cloneSession(s)
	return s, nil
}

// activity adds up the confirmed sales and the movements of a session.
func (f fakeCash) activity(id uuid.UUID) domain.CashActivity {
	a := domain.CashActivity{CashReceived: decimal.Zero, ChangeGiven: decimal.Zero, CashIn: decimal.Zero, CashOut: decimal.Zero}
	byMethod := map[domain.PaymentMethod]decimal.Decimal{}
	for _, sale := range f.sales {
		if sid := sale.CashSessionID(); sid == nil || *sid != id || sale.Status() != domain.SaleConfirmed {
			continue
		}
		a.Sales++
		a.ChangeGiven = a.ChangeGiven.Add(sale.Change())
		byMethod[domain.PaymentCash] = byMethod[domain.PaymentCash].Sub(sale.Change())
		for _, p := range sale.Payments() {
			if p.Method() == domain.PaymentCash {
				a.CashReceived = a.CashReceived.Add(p.Amount())
			}
			byMethod[p.Method()] = byMethod[p.Method()].Add(p.Amount())
		}
	}
	for _, m := range []domain.PaymentMethod{domain.PaymentCash, domain.PaymentCard, domain.PaymentNequi, domain.PaymentDaviplata, domain.PaymentTransfer} {
		if amount, ok := byMethod[m]; ok {
			a.ByMethod = append(a.ByMethod, domain.MethodTotal{Method: m, Amount: amount})
		}
	}
	for _, m := range f.movements {
		if m.SessionID() != id {
			continue
		}
		if m.Kind() == domain.CashIn {
			a.CashIn = a.CashIn.Add(m.Amount())
		} else {
			a.CashOut = a.CashOut.Add(m.Amount())
		}
	}
	return a
}
