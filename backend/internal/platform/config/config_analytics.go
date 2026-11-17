package config

import (
	"errors"
	"fmt"
	"net/url"
)

// minAnalyticsSecret is the shortest key accepted for the analytics
// pseudonyms (HMAC-SHA256).
const minAnalyticsSecret = 32

// loadAnalytics reads ANALYTICS_SECRET (the key of the users' pseudonyms in
// the usage analytics; empty turns the analytics off) and
// FEEDBACK_WEBHOOK_URL (the team's chat webhook for the users' comments;
// empty writes them in the log). Neither value is ever echoed.
func (cfg *Config) loadAnalytics(getenv func(string) string) error {
	var errs []error
	if secret := getenv("ANALYTICS_SECRET"); secret != "" {
		if len(secret) < minAnalyticsSecret {
			errs = append(errs, fmt.Errorf("ANALYTICS_SECRET must have at least %d bytes", minAnalyticsSecret))
		} else {
			cfg.AnalyticsSecret = []byte(secret)
		}
	}
	if raw := getenv("FEEDBACK_WEBHOOK_URL"); raw != "" {
		u, err := url.Parse(raw)
		switch {
		case err != nil || u.Host == "":
			errs = append(errs, errors.New("FEEDBACK_WEBHOOK_URL is not a URL"))
		case u.Scheme != "https" && cfg.Env == productionEnv:
			errs = append(errs, errors.New("FEEDBACK_WEBHOOK_URL must use https in production"))
		default:
			cfg.FeedbackWebhookURL = raw
		}
	}
	return errors.Join(errs...)
}
