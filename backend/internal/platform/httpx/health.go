package httpx

import "net/http"

// Healthz is the liveness probe: the process is alive.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz is the readiness probe: dependencies are OK. It is always ready until
// the database is wired in (T03).
func Readyz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
