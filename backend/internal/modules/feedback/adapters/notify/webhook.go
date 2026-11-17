package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
)

// webhookTimeout is how long a notice may take; a slow chat is retried.
const webhookTimeout = 10 * time.Second

// Webhook posts each notice as JSON to the team's chat (Slack, Discord or
// Google Chat incoming webhooks read "text"; Discord reads "content").
type Webhook struct {
	URL    string
	Client *http.Client
}

var _ app.Notifier = Webhook{}

// NewWebhook builds a Webhook to url.
func NewWebhook(url string) Webhook {
	return Webhook{URL: url, Client: &http.Client{Timeout: webhookTimeout}}
}

// Notify posts the notice; a status other than 2xx is an error, so the job
// is retried.
func (w Webhook) Notify(ctx context.Context, n app.Notice) error {
	text := fmt.Sprintf("Nuevo comentario en Cuadra (%s, versión %s, empresa %s):\n%s",
		n.Screen, n.AppVersion, n.TenantID, n.Message)
	body, err := json.Marshal(map[string]string{"text": text, "content": text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.Client.Do(req)
	if err != nil {
		return fmt.Errorf("feedback: webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("feedback: webhook answered %d", resp.StatusCode)
	}
	return nil
}
