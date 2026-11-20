package app

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// DismissChecklist hides the actor's checklist, remembering what was
// pending: it comes back only if a critical step goes missing afterwards.
// Hiding it again replaces the earlier dismissal.
func (s *Service) DismissChecklist(ctx context.Context, a Actor) (err error) {
	ctx, span := s.startSpan(ctx, "DismissChecklist", a)
	defer func() { usecase.End(span, err) }()

	in, err := s.checklistInput(ctx, a)
	if err != nil {
		return err
	}
	d, err := domain.NewDismissal(a.TenantID, a.UserID, domain.BuildChecklist(in), s.Now())
	if err != nil {
		return err
	}
	if err := s.Dismissals.Save(ctx, d); err != nil {
		return err
	}
	s.dismissals.Add(ctx, 1)
	return nil
}
