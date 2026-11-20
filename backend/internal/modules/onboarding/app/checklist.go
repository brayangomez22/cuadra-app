package app

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// invoicingPermission resolves the invoicing step: whoever lacks it never
// sees the step, so its readiness is not read for them.
const invoicingPermission = "invoicing:manage"

// GetChecklist returns the actor's getting-started checklist, computed from
// the tenant's data at this moment.
func (s *Service) GetChecklist(ctx context.Context, a Actor) (_ domain.Checklist, err error) {
	ctx, span := s.startSpan(ctx, "GetChecklist", a)
	defer func() { usecase.End(span, err) }()

	in, err := s.checklistInput(ctx, a)
	if err != nil {
		return domain.Checklist{}, err
	}
	d, ok, err := s.Dismissals.Get(ctx, a.TenantID, a.UserID)
	if err != nil {
		return domain.Checklist{}, err
	}
	if ok {
		in.Dismissal = &d
	}
	return domain.BuildChecklist(in), nil
}

// checklistInput gathers what the checklist is built from, without the
// user's dismissal.
func (s *Service) checklistInput(ctx context.Context, a Actor) (domain.ChecklistInput, error) {
	state, err := s.State.TenantState(ctx, a.TenantID)
	if err != nil {
		return domain.ChecklistInput{}, err
	}
	steps, _ := s.Templates.Steps(state.Template)
	modules, err := s.Modules.Active(ctx, a.TenantID)
	if err != nil {
		return domain.ChecklistInput{}, err
	}
	if s.Can(a.Role, invoicingPermission) {
		r, err := s.Invoicing.Readiness(ctx, a)
		if err != nil {
			return domain.ChecklistInput{}, err
		}
		state.Facts.InvoicingEnabled, state.Facts.InvoicingReady = r.Enabled, r.Ready
	}
	return domain.ChecklistInput{
		Steps: steps, Facts: state.Facts, Modules: modules,
		Can: func(p string) bool { return s.Can(a.Role, p) },
	}, nil
}
