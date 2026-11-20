package app

import (
	"context"
	"errors"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Readiness is whether a tenant can issue electronic documents now:
// emission turned on, and a resolution of its default kind that would
// number a document today (domain.PickResolution).
type Readiness struct {
	Enabled bool
	Ready   bool
}

// Readiness tells the getting-started checklist whether the tenant's
// electronic invoicing is ready, with the same rule a sale's document uses
// to pick its resolution.
func (s *Service) Readiness(ctx context.Context, a Actor) (_ Readiness, err error) {
	ctx, span := s.startSpan(ctx, "Readiness", a)
	defer func() { usecase.End(span, err) }()

	settings, err := s.Settings.Get(ctx, a.TenantID)
	if err != nil || !settings.Enabled {
		return Readiness{}, err
	}
	rs, err := s.Resolutions.List(ctx, a.TenantID)
	if err != nil {
		return Readiness{}, err
	}
	_, err = domain.PickResolution(rs, settings.DefaultKind, s.Now())
	if errors.Is(err, domain.ErrNoActiveResolution) {
		return Readiness{Enabled: true}, nil
	}
	if err != nil {
		return Readiness{}, err
	}
	return Readiness{Enabled: true, Ready: true}, nil
}
