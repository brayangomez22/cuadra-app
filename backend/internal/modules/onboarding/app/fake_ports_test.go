package app_test

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

type fakeState struct {
	template string
	facts    domain.Facts
}

func (f *fakeState) TenantState(context.Context, uuid.UUID) (domain.TenantState, error) {
	return domain.TenantState{Template: f.template, Facts: f.facts}, nil
}

type fakeTemplates struct{}

func (fakeTemplates) Steps(id string) ([]string, bool) {
	steps, ok := templateSteps[id]
	return steps, ok
}

type fakeInvoicing struct{ readiness app.InvoicingReadiness }

func (f *fakeInvoicing) Readiness(context.Context, app.Actor) (app.InvoicingReadiness, error) {
	return f.readiness, nil
}

type dismissalKey struct{ tenant, user uuid.UUID }

type fakeDismissals struct {
	saved map[dismissalKey]domain.Dismissal
}

func newFakeDismissals() *fakeDismissals {
	return &fakeDismissals{saved: map[dismissalKey]domain.Dismissal{}}
}

func (f *fakeDismissals) Get(_ context.Context, tenantID, userID uuid.UUID) (domain.Dismissal, bool, error) {
	d, ok := f.saved[dismissalKey{tenantID, userID}]
	return d, ok, nil
}

func (f *fakeDismissals) Save(_ context.Context, d domain.Dismissal) error {
	f.saved[dismissalKey{d.TenantID(), d.UserID()}] = d
	return nil
}
