package httpadapter_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

func TestListAuditEvents(t *testing.T) {
	t.Run("traduce los filtros y responde la página", func(t *testing.T) {
		entity, user := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		e, err := domain.NewEvent(domain.EventInput{
			ID: uuid.Must(uuid.NewV7()), TenantID: owner.TenantID, UserID: user, Role: "cashier",
			Action: domain.ActionSaleVoided, EntityID: &entity,
			OccurredAt: time.Date(2026, 11, 15, 14, 0, 0, 0, time.UTC), Details: map[string]string{"total": "52000.00"},
		})
		require.NoError(t, err)
		uc := &stubUseCases{page: domain.EventPage{Items: []*domain.Event{e}, Total: 7}}

		rec := serve(t, uc, "/api/v1/audit-events?action=sale.voided&user_id="+user.String()+"&from=2026-11-01&to=2026-11-15&limit=5&offset=10")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"total": 7, "items": [{
			"id": "`+e.ID().String()+`", "action": "sale.voided", "user_id": "`+user.String()+`", "role": "cashier",
			"entity_type": "sale", "entity_id": "`+entity.String()+`", "occurred_at": "2026-11-15T14:00:00Z",
			"details": {"total": "52000.00"}}]}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, "owner", uc.actor.Role)
		require.Equal(t, "sale.voided", uc.query.Action)
		require.Equal(t, &user, uc.query.UserID)
		require.Equal(t, "2026-11-01", uc.query.From.Format(time.DateOnly))
		require.Equal(t, "2026-11-15", uc.query.To.Format(time.DateOnly))
		require.Equal(t, 5, uc.query.Limit)
		require.Equal(t, 10, uc.query.Offset)
	})

	t.Run("sin límite usa el del contrato", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, "/api/v1/audit-events")
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"items": [], "total": 0}`, rec.Body.String())
		require.Equal(t, 20, uc.query.Limit)
	})

	t.Run("un rango invertido responde 422", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrInvalidFilter}, "/api/v1/audit-events?from=2026-11-15&to=2026-11-01")
		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_filter")
	})
}
