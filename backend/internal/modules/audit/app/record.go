package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Entry is what an audited operation did: an action of the catalog, the
// entity it is about (nil for a module), where, and the details the action
// declares.
type Entry struct {
	Action     domain.Action
	EntityID   *uuid.UUID
	LocationID *uuid.UUID
	Details    map[string]string
}

// Record appends e to the actor's audit log, at the current time. The
// other modules call it inside the transaction of the audited operation
// (with a ctx from db.InTenantTx): an error must undo the operation, so a
// change is never stored without its event, nor an event without its
// change.
func (s *Service) Record(ctx context.Context, a Actor, e Entry) (err error) {
	ctx, span := s.startSpan(ctx, "Record", a, attribute.String("audit.action", string(e.Action)))
	defer func() { usecase.End(span, err) }()

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("audit: new event id: %w", err)
	}
	event, err := domain.NewEvent(domain.EventInput{
		ID: id, TenantID: a.TenantID, UserID: a.UserID, Role: a.Role,
		Action: e.Action, EntityID: e.EntityID, LocationID: e.LocationID,
		OccurredAt: s.Now(), Details: e.Details,
	})
	if err != nil {
		return err
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return err
	}
	s.events.Add(ctx, 1, metric.WithAttributes(attribute.String("action", string(e.Action))))
	return nil
}
