// Package notify tells the team about a comment: by a webhook of its chat
// (FEEDBACK_WEBHOOK_URL) or, when none is configured, in the log.
package notify

import (
	"context"
	"log/slog"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
)

// Log writes a notice in the log, without the comment's text (the user may
// have written personal data): the team reads it in the database.
type Log struct{ Logger *slog.Logger }

var _ app.Notifier = Log{}

// Notify logs that a comment arrived.
func (l Log) Notify(ctx context.Context, n app.Notice) error {
	l.Logger.InfoContext(ctx, "feedback notice",
		slog.String("feedback_id", n.FeedbackID.String()), slog.String("screen", n.Screen),
		slog.String("app_version", n.AppVersion))
	return nil
}
