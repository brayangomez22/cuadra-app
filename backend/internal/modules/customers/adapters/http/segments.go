package httpadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http/customersapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// SegmentUseCases are the use cases of the customer segments.
type SegmentUseCases interface {
	CreateSegment(ctx context.Context, a app.Actor, name string) (*domain.Segment, error)
	RenameSegment(ctx context.Context, a app.Actor, id uuid.UUID, name string) (*domain.Segment, error)
	DeleteSegment(ctx context.Context, a app.Actor, id uuid.UUID) error
	ListSegments(ctx context.Context, a app.Actor) ([]*domain.Segment, error)
}

// segmentHandlers are the operations on customer segments.
type segmentHandlers struct{ uc SegmentUseCases }

// errSegmentPathNotFound is domain.ErrSegmentNotFound when the segment is
// the resource of the path (404). As the segment of a customer it is a
// field of the body, and stays a 422.
var errSegmentPathNotFound = errors.New("customers: segment of the path not found")

// segmentPathError turns a missing segment into errSegmentPathNotFound.
func segmentPathError(err error) error {
	if errors.Is(err, domain.ErrSegmentNotFound) {
		return fmt.Errorf("%w: %w", errSegmentPathNotFound, err)
	}
	return err
}

// ListCustomerSegments lists the tenant's segments.
func (h *segmentHandlers) ListCustomerSegments(ctx context.Context, _ customersapi.ListCustomerSegmentsRequestObject) (customersapi.ListCustomerSegmentsResponseObject, error) {
	list, err := h.uc.ListSegments(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]customersapi.CustomerSegment, 0, len(list))
	for _, s := range list {
		items = append(items, segmentBody(s))
	}
	return customersapi.ListCustomerSegments200JSONResponse{Items: items}, nil
}

// CreateCustomerSegment adds a segment.
func (h *segmentHandlers) CreateCustomerSegment(ctx context.Context, req customersapi.CreateCustomerSegmentRequestObject) (customersapi.CreateCustomerSegmentResponseObject, error) {
	s, err := h.uc.CreateSegment(ctx, actor(ctx), req.Body.Name)
	if err != nil {
		return nil, err
	}
	return customersapi.CreateCustomerSegment201JSONResponse(segmentBody(s)), nil
}

// RenameCustomerSegment changes a segment's name.
func (h *segmentHandlers) RenameCustomerSegment(ctx context.Context, req customersapi.RenameCustomerSegmentRequestObject) (customersapi.RenameCustomerSegmentResponseObject, error) {
	s, err := h.uc.RenameSegment(ctx, actor(ctx), req.SegmentId, req.Body.Name)
	if err != nil {
		return nil, segmentPathError(err)
	}
	return customersapi.RenameCustomerSegment200JSONResponse(segmentBody(s)), nil
}

// DeleteCustomerSegment removes a segment without customers.
func (h *segmentHandlers) DeleteCustomerSegment(ctx context.Context, req customersapi.DeleteCustomerSegmentRequestObject) (customersapi.DeleteCustomerSegmentResponseObject, error) {
	if err := h.uc.DeleteSegment(ctx, actor(ctx), req.SegmentId); err != nil {
		return nil, segmentPathError(err)
	}
	return customersapi.DeleteCustomerSegment204Response{}, nil
}

func segmentBody(s *domain.Segment) customersapi.CustomerSegment {
	return customersapi.CustomerSegment{Id: s.ID(), Name: s.Name()}
}
