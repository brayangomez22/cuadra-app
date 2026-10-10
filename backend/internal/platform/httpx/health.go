package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// readyTimeout bounds each readiness check, so a hung database fails the
// probe instead of blocking it.
const readyTimeout = 2 * time.Second

// Pinger is a dependency the API needs to serve traffic.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Healthz is the liveness probe: the process is alive.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz is the readiness probe: it answers 503 while the database is
// unreachable, so traffic is routed elsewhere without restarting the process.
func Readyz(log *slog.Logger, db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			log.WarnContext(ctx, "readiness check failed", slog.String("dependency", "database"), slog.Any("error", err))
			WriteError(w, http.StatusServiceUnavailable, "not_ready", "El servicio no está listo. Intenta de nuevo en unos segundos.")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
