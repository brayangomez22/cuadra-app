package httpx

import (
	"context"
	"log/slog"
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx/healthapi"
)

// readyTimeout bounds each readiness check, so a hung database fails the
// probe instead of blocking it.
const readyTimeout = 2 * time.Second

// Pinger is a dependency the API needs to serve traffic.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health implements the generated health API (operations tagged "health").
type Health struct {
	log *slog.Logger
	db  Pinger
}

var _ healthapi.StrictServerInterface = (*Health)(nil)

// NewHealth builds the health handlers; db backs the readiness probe.
func NewHealth(log *slog.Logger, db Pinger) *Health {
	return &Health{log: log, db: db}
}

// GetHealthz is the liveness probe: the process is alive.
func (h *Health) GetHealthz(context.Context, healthapi.GetHealthzRequestObject) (healthapi.GetHealthzResponseObject, error) {
	return healthapi.GetHealthz200JSONResponse{Status: healthapi.Ok}, nil
}

// GetReadyz is the readiness probe: it answers 503 while the database is
// unreachable, so traffic is routed elsewhere without restarting the process.
func (h *Health) GetReadyz(ctx context.Context, _ healthapi.GetReadyzRequestObject) (healthapi.GetReadyzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		h.log.WarnContext(ctx, "readiness check failed", slog.String("dependency", "database"), slog.Any("error", err))
		return healthapi.GetReadyz503JSONResponse{Error: healthapi.ErrorDetail{
			Code:    "not_ready",
			Message: "El servicio no está listo. Intenta de nuevo en unos segundos.",
		}}, nil
	}
	return healthapi.GetReadyz200JSONResponse{Status: healthapi.Ready}, nil
}
