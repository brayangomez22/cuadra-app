package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// invalidTextMessage is the answer to a text the API cannot store.
const invalidTextMessage = "La solicitud tiene caracteres no válidos."

// RejectNUL answers 400 invalid_request to a request whose query or JSON
// body carries a NUL character: PostgreSQL cannot store it in a text
// column, and the request would otherwise fail deep in a repository with a
// 500 (found by the DAST scan, T38). A body is decoded only when its bytes
// contain the escape \u0000; a multipart body (an import's file) is not
// read. It runs after LimitBody, so the body it reads is bounded.
func RejectNUL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if queryHasNUL(r) {
			WriteError(w, http.StatusBadRequest, "invalid_request", invalidTextMessage)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Over LimitBody's cap or a broken connection: the next layers
			// answer it as they would.
			r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), errReader{err}))
			next.ServeHTTP(w, r)
			return
		}
		if hasNUL(body) {
			WriteError(w, http.StatusBadRequest, "invalid_request", invalidTextMessage)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

// queryHasNUL tells whether a key or a value of the query has a NUL.
func queryHasNUL(r *http.Request) bool {
	for k, values := range r.URL.Query() {
		if strings.ContainsRune(k, 0) || containsNUL(toAny(values)) {
			return true
		}
	}
	return false
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// hasNUL tells whether a JSON document has a NUL in a key or a string.
func hasNUL(body []byte) bool {
	if !bytes.Contains(bytes.ToLower(body), []byte(`\u0000`)) {
		return false
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return false // the validator rejects malformed JSON
	}
	return containsNUL(doc)
}

func containsNUL(v any) bool {
	switch v := v.(type) {
	case string:
		return strings.ContainsRune(v, 0)
	case []any:
		for _, item := range v {
			if containsNUL(item) {
				return true
			}
		}
	case map[string]any:
		for k, item := range v {
			if strings.ContainsRune(k, 0) || containsNUL(item) {
				return true
			}
		}
	}
	return false
}

// errReader returns err once its reader is drained.
type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
