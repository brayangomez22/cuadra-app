package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Generic user-facing messages for errors whose details must stay internal.
const (
	InvalidRequestMessage = "La solicitud no es válida. Revisa los datos enviados."
	InternalErrorMessage  = "Ocurrió un error inesperado. Intenta de nuevo."
)

// ErrorRule maps a domain error to its HTTP status, its stable code and a
// Spanish message the user can read. Err is matched with errors.Is, so
// wrapped errors are mapped too.
type ErrorRule struct {
	Err     error
	Status  int
	Code    string
	Message string
}

// ErrorMapper translates the errors returned by a module's handlers into the
// standard Error schema. Its methods match the RequestErrorHandlerFunc and
// ResponseErrorHandlerFunc options of the generated strict servers.
type ErrorMapper struct {
	log   *slog.Logger
	rules []ErrorRule
}

// NewErrorMapper builds a mapper with the given rules. Rules are checked in
// order; the first match wins.
func NewErrorMapper(log *slog.Logger, rules ...ErrorRule) *ErrorMapper {
	return &ErrorMapper{log: log, rules: rules}
}

// RequestError answers 400 when the generated code cannot decode a request
// (malformed JSON, unparsable parameter). The decoder's message is logged,
// not sent: it is in English and may describe internals.
func (m *ErrorMapper) RequestError(w http.ResponseWriter, r *http.Request, err error) {
	m.log.InfoContext(r.Context(), "request rejected", slog.Any("error", err))
	WriteError(w, http.StatusBadRequest, "invalid_request", InvalidRequestMessage)
}

// ResponseError answers with the rule matching err, or with 500 when no rule
// matches. Unmapped errors are bugs or infrastructure failures: they are
// logged and recorded on the request span, but never sent to the client.
func (m *ErrorMapper) ResponseError(w http.ResponseWriter, r *http.Request, err error) {
	for _, rule := range m.rules {
		if errors.Is(err, rule.Err) {
			WriteError(w, rule.Status, rule.Code, rule.Message)
			return
		}
	}

	span := trace.SpanFromContext(r.Context())
	span.RecordError(err)
	span.SetStatus(codes.Error, "unhandled error")
	m.log.ErrorContext(r.Context(), "unhandled error", slog.Any("error", err))
	WriteError(w, http.StatusInternalServerError, "internal_error", InternalErrorMessage)
}
