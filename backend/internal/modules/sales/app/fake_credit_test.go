package app_test

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// fakeCredit is the receivables module: it records the charges and the
// cancellations; chargeErr and cancelErr, when set, make them fail.
type fakeCredit struct {
	mu        sync.Mutex
	charges   []app.CreditCharge
	canceled  []uuid.UUID
	chargeErr error
	cancelErr error
}

func (f *fakeCredit) Charge(_ context.Context, c app.CreditCharge) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.chargeErr != nil {
		return f.chargeErr
	}
	f.charges = append(f.charges, c)
	return nil
}

func (f *fakeCredit) CancelSale(_ context.Context, _, _, saleID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancelErr != nil {
		return f.cancelErr
	}
	f.canceled = append(f.canceled, saleID)
	return nil
}

// fakeAuthorizer knows the users allowed to exceed a customer's credit, by
// their credentials; calls counts how many times it was asked.
type fakeAuthorizer struct {
	mu    sync.Mutex
	users map[app.Credentials]uuid.UUID
	calls int
}

func (f *fakeAuthorizer) AuthorizeCredit(_ context.Context, _ uuid.UUID, c app.Credentials) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if id, ok := f.users[c]; ok {
		return id, nil
	}
	return uuid.Nil, domain.ErrCreditAuthorizationRejected
}
