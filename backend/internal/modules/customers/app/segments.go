package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CreateSegment stores a new customer segment.
func (s *Service) CreateSegment(ctx context.Context, a Actor, name string) (_ *domain.Segment, err error) {
	ctx, span := s.startSpan(ctx, "CreateSegment", a)
	defer func() { usecase.End(span, err) }()

	seg, err := domain.NewSegment(a.TenantID, name, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(segmentAttr(seg.ID()))
	if err := s.Segments.Create(ctx, seg); err != nil {
		return nil, err
	}
	return seg, nil
}

// RenameSegment changes a segment's name; its customers stay in it.
func (s *Service) RenameSegment(ctx context.Context, a Actor, id uuid.UUID, name string) (_ *domain.Segment, err error) {
	ctx, span := s.startSpan(ctx, "RenameSegment", a, segmentAttr(id))
	defer func() { usecase.End(span, err) }()

	seg, err := s.Segments.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := seg.Rename(name); err != nil {
		return nil, err
	}
	if err := s.Segments.Update(ctx, seg); err != nil {
		return nil, err
	}
	return seg, nil
}

// DeleteSegment removes a segment no customer is in.
func (s *Service) DeleteSegment(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteSegment", a, segmentAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Segments.Delete(ctx, a.TenantID, id)
}

// ListSegments returns the tenant's segments, by name.
func (s *Service) ListSegments(ctx context.Context, a Actor) (_ []*domain.Segment, err error) {
	ctx, span := s.startSpan(ctx, "ListSegments", a)
	defer func() { usecase.End(span, err) }()

	return s.Segments.List(ctx, a.TenantID)
}

// SeedSegments stores the segments of a new tenant's business template.
// Called with the sign-up's context, its writes join the sign-up's
// transaction; no user exists yet.
func (s *Service) SeedSegments(ctx context.Context, tenantID uuid.UUID, names []string) (err error) {
	ctx, span := s.startSpan(ctx, "SeedSegments", Actor{TenantID: tenantID})
	defer func() { usecase.End(span, err) }()

	for _, name := range names {
		seg, err := domain.NewSegment(tenantID, name, s.Now())
		if err != nil {
			return err
		}
		if err := s.Segments.Create(ctx, seg); err != nil {
			return err
		}
	}
	return nil
}
