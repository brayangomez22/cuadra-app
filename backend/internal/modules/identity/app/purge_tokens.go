package app

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// purgeTenantsPerPass bounds the tenants read in one pass of the purge.
const purgeTenantsPerPass = 500

// PurgeExpiredRefreshTokens deletes the refresh tokens of every tenant that
// expired (P07): they open no session, and a family whose last token
// expired cannot be reused. It works tenant by tenant, each within its own
// Row-Level Security, and returns how many it deleted. It runs every night
// (identity.purge_refresh_tokens); a failure leaves the rest for the next
// run.
func (s *Service) PurgeExpiredRefreshTokens(ctx context.Context) (_ int, err error) {
	ctx, span := s.startSpan(ctx, "PurgeExpiredRefreshTokens")
	defer func() { usecase.End(span, err) }()

	before := s.Now()
	var purged int64
	for {
		tenants, err := s.ExpiredTokens.TenantsWithExpired(ctx, before, purgeTenantsPerPass)
		if err != nil {
			return int(purged), err
		}
		for _, tenantID := range tenants {
			n, err := s.ExpiredTokens.DeleteExpired(ctx, tenantID, before)
			if err != nil {
				return int(purged), err
			}
			purged += n
			s.tokensPurged.Add(ctx, n)
		}
		if len(tenants) < purgeTenantsPerPass {
			break
		}
	}
	span.SetAttributes(attribute.Int64("tokens.purged", purged))
	s.Logger.InfoContext(ctx, "expired refresh tokens purged", slog.Int64("tokens", purged))
	return int(purged), nil
}
