package app

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// FeedbackInput is what the user sends: the comment, the screen it was on
// and the version of the app.
type FeedbackInput struct {
	Message    string
	Screen     string
	AppVersion string
}

// SendFeedback stores a user's comment and queues the notice to the team in
// the same transaction. Any signed-in user may send one.
func (s *Service) SendFeedback(ctx context.Context, a Actor, in FeedbackInput) (_ *domain.Feedback, err error) {
	ctx, span := s.startSpan(ctx, "SendFeedback", a.TenantID, attribute.String("user.id", a.UserID.String()))
	defer func() { usecase.End(span, err) }()

	f, err := domain.NewFeedback(a.TenantID, a.UserID, in.Message, in.Screen, in.AppVersion, s.Now())
	if err != nil {
		return nil, err
	}
	if err := s.Feedback.Create(ctx, f); err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.String("feedback.id", f.ID().String()))
	s.received.Add(ctx, 1)
	s.Logger.InfoContext(ctx, "feedback received", slog.String("feedback_id", f.ID().String()))
	return f, nil
}
