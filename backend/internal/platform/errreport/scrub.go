package errreport

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/getsentry/sentry-go"
)

// filtered replaces the value of a sensitive key.
const filtered = "[filtered]"

// textRules remove personal data and secrets from free text: messages, error
// strings, breadcrumbs. Order matters: the connection-string rule runs before
// the email one, which would otherwise eat "user:password@host" partially.
var textRules = []struct {
	re   *regexp.Regexp
	repl string
}{
	// postgres://user:password@host → postgres://[filtered]@host
	{regexp.MustCompile(`://[^/\s:@]+:[^/\s@]+@`), "://" + filtered + "@"},
	{regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`), "[email]"},
	{regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=\-]+`), "Bearer [token]"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]*`), "[token]"},
	// Refresh token: <tenant uuid>.<32 random bytes in base64url>.
	{regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\.[A-Za-z0-9_\-]{32,}`), "[token]"},
	{regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|api[_-]?key)(\s*[=:]\s*)\S+`), "${1}${2}" + filtered},
}

// sensitiveKeys are substrings of map keys whose values are never sent.
var sensitiveKeys = []string{"password", "passwd", "secret", "token", "authorization", "cookie", "dsn", "apikey", "api_key", "email"}

// allowedHeaders are the only request headers kept; the rest may carry
// credentials (Authorization, Cookie, custom API keys) or personal data.
var allowedHeaders = map[string]bool{"User-Agent": true, "Content-Type": true, "Accept": true}

// Scrub is the BeforeSend hook: it removes personal data and secrets from an
// event before it leaves the process. It keeps only the user's opaque ID, and
// of the request only its method, path and harmless headers.
func Scrub(e *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	e.Message = scrubText(e.Message)
	for i := range e.Exception {
		e.Exception[i].Value = scrubText(e.Exception[i].Value)
	}
	for k, v := range e.Tags {
		e.Tags[k] = scrubText(v)
	}
	for name, c := range e.Contexts {
		e.Contexts[name] = scrubMap(c)
	}
	for _, b := range e.Breadcrumbs {
		b.Message = scrubText(b.Message)
		b.Data = scrubMap(b.Data)
	}
	e.User = sentry.User{ID: e.User.ID}
	if r := e.Request; r != nil {
		r.URL = stripQuery(r.URL)
		r.Data, r.QueryString, r.Cookies, r.Env = "", "", "", nil
		headers := map[string]string{}
		for k, v := range r.Headers {
			if allowedHeaders[k] {
				headers[k] = v
			}
		}
		r.Headers = headers
	}
	return e
}

func scrubText(s string) string {
	for _, rule := range textRules {
		s = rule.re.ReplaceAllString(s, rule.repl)
	}
	return s
}

// scrubMap filters the values of sensitive keys and scrubs the strings and
// nested maps of the rest. A URL loses its query string.
func scrubMap(m map[string]any) map[string]any {
	for k, v := range m {
		if isSensitiveKey(k) {
			m[k] = filtered
			continue
		}
		switch v := v.(type) {
		case string:
			m[k] = scrubText(stripQuery(v))
		case map[string]any:
			m[k] = scrubMap(v)
		}
	}
	return m
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

// stripQuery drops the query string and fragment of a URL-like value; other
// strings are returned as they are.
func stripQuery(s string) string {
	if !strings.HasPrefix(s, "/") && !strings.Contains(s, "://") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	u.RawQuery, u.Fragment = "", ""
	return u.String()
}
