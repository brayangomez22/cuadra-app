package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/http/auditapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// EventUseCases are the use cases of the audit log (*app.Service).
type EventUseCases interface {
	ListEvents(ctx context.Context, a app.Actor, q app.EventQuery) (domain.EventPage, error)
}

type eventHandlers struct{ uc EventUseCases }

// ListAuditEvents returns a page of the tenant's audit log.
func (h *eventHandlers) ListAuditEvents(ctx context.Context, req auditapi.ListAuditEventsRequestObject) (auditapi.ListAuditEventsResponseObject, error) {
	p := req.Params
	q := app.EventQuery{UserID: p.UserId, EntityID: p.EntityId, Limit: defaultPageSize}
	if p.Action != nil {
		q.Action = string(*p.Action)
	}
	if p.From != nil {
		q.From = &p.From.Time
	}
	if p.To != nil {
		q.To = &p.To.Time
	}
	if p.Limit != nil {
		q.Limit = *p.Limit
	}
	if p.Offset != nil {
		q.Offset = *p.Offset
	}
	page, err := h.uc.ListEvents(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	items := make([]auditapi.AuditEvent, 0, len(page.Items))
	for _, e := range page.Items {
		items = append(items, eventBody(e))
	}
	return auditapi.ListAuditEvents200JSONResponse{Items: items, Total: page.Total}, nil
}

func eventBody(e *domain.Event) auditapi.AuditEvent {
	return auditapi.AuditEvent{
		Id: e.ID(), Action: auditapi.AuditAction(e.Action()), UserId: e.UserID(), Role: e.Role(),
		EntityType: e.EntityType(), EntityId: e.EntityID(), LocationId: e.LocationID(),
		OccurredAt: e.OccurredAt(), Details: e.Details(),
	}
}
