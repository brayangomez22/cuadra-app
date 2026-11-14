package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ResolutionView is a resolution and whether it is about to end today.
type ResolutionView struct {
	Resolution *domain.Resolution
	NearEnd    bool
}

// CreateResolution registers a numbering resolution the DIAN authorized; it
// cannot overlap another of the same kind and prefix
// (domain.ErrResolutionOverlaps), so no number is ever issued twice.
func (s *Service) CreateResolution(ctx context.Context, a Actor, p domain.ResolutionParams) (_ ResolutionView, err error) {
	ctx, span := s.startSpan(ctx, "CreateResolution", a)
	defer func() { usecase.End(span, err) }()

	now := s.Now()
	r, err := domain.NewResolution(uuid.Must(uuid.NewV7()), a.TenantID, p, now)
	if err != nil {
		return ResolutionView{}, err
	}
	existing, err := s.Resolutions.List(ctx, a.TenantID)
	if err != nil {
		return ResolutionView{}, err
	}
	for _, o := range existing {
		if r.Overlaps(o) {
			return ResolutionView{}, domain.ErrResolutionOverlaps
		}
	}
	if err := s.Resolutions.Create(ctx, r); err != nil {
		return ResolutionView{}, err
	}
	return ResolutionView{Resolution: r, NearEnd: r.NearEnd(now)}, nil
}

// ListResolutions returns the tenant's resolutions, the newest first, with
// whether each is about to end.
func (s *Service) ListResolutions(ctx context.Context, a Actor) (_ []ResolutionView, err error) {
	ctx, span := s.startSpan(ctx, "ListResolutions", a)
	defer func() { usecase.End(span, err) }()

	rs, err := s.Resolutions.List(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	views := make([]ResolutionView, len(rs))
	for i, r := range rs {
		views[i] = ResolutionView{Resolution: r, NearEnd: r.NearEnd(now)}
	}
	return views, nil
}

// DeleteResolution removes a resolution registered by mistake, before it
// numbers any document (domain.ErrResolutionInUse).
func (s *Service) DeleteResolution(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteResolution", a)
	defer func() { usecase.End(span, err) }()

	return s.Resolutions.Delete(ctx, a.TenantID, id)
}
