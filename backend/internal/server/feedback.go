package server

import "github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback"

// feedback builds the module of the users' comments, whose notices to the
// team go by a background job.
func (w *wiring) feedback() (*feedback.Module, error) {
	return feedback.New(feedback.Config{
		DB:             w.DB,
		Jobs:           w.Jobs,
		Workers:        w.Workers,
		WebhookURL:     w.FeedbackWebhookURL,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}
