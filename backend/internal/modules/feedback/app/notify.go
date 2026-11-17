package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// NotifyFeedback tells the team about a comment, once: a comment already
// notified, or one that no longer exists, is done. A failed notice returns
// its error so the job is retried.
func (s *Service) NotifyFeedback(ctx context.Context, tenantID, feedbackID uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "NotifyFeedback", tenantID, attribute.String("feedback.id", feedbackID.String()))
	defer func() { usecase.End(span, err) }()

	f, err := s.Feedback.Get(ctx, tenantID, feedbackID)
	switch {
	case errors.Is(err, domain.ErrFeedbackMissing):
		return nil
	case err != nil:
		return err
	case f.Notified():
		return nil
	}
	err = s.Notifier.Notify(ctx, Notice{
		FeedbackID: f.ID(), TenantID: f.TenantID(), Screen: f.Screen(), AppVersion: f.AppVersion(),
		Message: f.Message(), CreatedAt: f.CreatedAt(),
	})
	if err != nil {
		s.notifications.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "failed")))
		return err
	}
	f.MarkNotified(s.Now())
	if err := s.Feedback.MarkNotified(ctx, f); err != nil {
		return err
	}
	s.notifications.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "sent")))
	return nil
}
