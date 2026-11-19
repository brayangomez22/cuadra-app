package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Receivables is the receivables module's aging, through a port
// internal/server adapts: the reports do not repeat its rules.
type Receivables interface {
	// Aging returns every customer's aging as of today, in sort's order
	// (overdue, total or over_90).
	Aging(ctx context.Context, tenantID, userID uuid.UUID, sort string) (domain.AgingExport, error)
}

// ReceivablesAging returns the whole aging, for its export.
func (s *Service) ReceivablesAging(ctx context.Context, a Actor, sort string) (_ domain.AgingExport, err error) {
	ctx, span := s.startSpan(ctx, "ReceivablesAging", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "receivables", time.Now())

	return s.Receivables.Aging(ctx, a.TenantID, a.UserID, sort)
}
