package httpx

import "net/http"

// LimitBody rejects with 413 a request whose declared body is over max
// bytes, before anything reads it, and caps the reading of a body without a
// declared length at max, so no request makes the API hold more than max
// bytes of it.
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				WriteError(w, http.StatusRequestEntityTooLarge, "request_too_large",
					"La solicitud es demasiado grande.")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
