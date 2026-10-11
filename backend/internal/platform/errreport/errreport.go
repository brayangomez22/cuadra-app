// Package errreport sends the application's errors to Sentry.
//
// Every log record at level Error becomes a Sentry event: the logging
// convention already says Error means "needs action", so panics recovered by
// the HTTP middleware, unmapped errors answered with 500 and failed jobs reach
// Sentry without each call site knowing about it. Only an allowlist leaves the
// process: the message, the exception built from the record's "error" or
// "panic" attribute, the tenant and request tags, the user's opaque ID and the
// trace context. The record's other attributes stay in the logs, and every
// event goes through Scrub before it is sent.
//
// Sentry is for errors only: traces, metrics and logs stay in OpenTelemetry.
// Without a DSN the reporter does nothing, so local runs and tests need no
// Sentry project.
package errreport

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// Options configures the reporter.
type Options struct {
	// DSN of the Sentry project; empty disables reporting.
	DSN         string
	Environment string
	Release     string
	// Extractors return the request's attributes from the context (the same
	// ones the logger uses). Only tenant_id and request_id become tags and
	// user_id becomes the user's ID; any other attribute is ignored.
	Extractors []func(context.Context) []slog.Attr
	// Transport replaces Sentry's HTTP transport; tests inject a fake one.
	Transport sentry.Transport
}

// Reporter sends error log records to Sentry. The zero value, or one built
// without a DSN, reports nothing.
type Reporter struct {
	client     *sentry.Client
	extractors []func(context.Context) []slog.Attr
}

// New builds a reporter. An invalid DSN is an error, so a misconfigured
// deployment fails at startup instead of losing its errors silently.
func New(opts Options) (*Reporter, error) {
	if opts.DSN == "" {
		return &Reporter{}, nil
	}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         opts.DSN,
		Environment: opts.Environment,
		Release:     opts.Release,
		BeforeSend:  Scrub,
		Transport:   opts.Transport,
		// No DataCollection: it only governs the SDK's HTTP auto-instrumentation,
		// which is not used. The event carries only what the handler puts in it.
	})
	if err != nil {
		return nil, errors.New("errreport: invalid Sentry configuration") // The DSN is not echoed.
	}
	return &Reporter{client: client, extractors: opts.Extractors}, nil
}

// Handler wraps next so records at level Error are also sent to Sentry. When
// reporting is disabled it returns next itself.
func (r *Reporter) Handler(next slog.Handler) slog.Handler {
	if r.client == nil {
		return next
	}
	return &handler{next: next, reporter: r}
}

// Flush waits up to timeout for pending events to be sent. It reports whether
// they all were.
func (r *Reporter) Flush(timeout time.Duration) bool {
	if r.client == nil {
		return true
	}
	return r.client.Flush(timeout)
}
