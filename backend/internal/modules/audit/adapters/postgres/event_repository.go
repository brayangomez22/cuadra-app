// Package postgres implements the audit log on PostgreSQL. Every query runs
// inside db.WithTenantTx, so Row-Level Security scopes it to the tenant;
// with a ctx from db.InTenantTx an append joins the audited operation's
// transaction.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// EventRepository implements domain.EventRepository. It only inserts and
// reads: app_user has no UPDATE nor DELETE on audit_events.
type EventRepository struct{ db *db.DB }

// NewEventRepository returns an EventRepository on d.
func NewEventRepository(d *db.DB) *EventRepository { return &EventRepository{db: d} }

// Append stores e; with a ctx from db.InTenantTx it joins that transaction.
func (r *EventRepository) Append(ctx context.Context, e *domain.Event) error {
	s := e.Snapshot()
	details, err := json.Marshal(s.Details)
	if err != nil {
		return fmt.Errorf("audit: encode details: %w", err)
	}
	err = r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).InsertEvent(ctx, sqlcgen.InsertEventParams{
			ID: s.ID, TenantID: s.TenantID, UserID: s.UserID, Role: s.Role,
			Action: string(s.Action), EntityType: e.EntityType(), EntityID: s.EntityID,
			LocationID: s.LocationID, OccurredAt: s.OccurredAt, Details: details,
		})
	})
	if err != nil {
		return fmt.Errorf("audit: append event: %w", err)
	}
	return nil
}

// List returns a page of the tenant's events matching f, newest first.
func (r *EventRepository) List(ctx context.Context, f domain.EventFilter) (domain.EventPage, error) {
	var action *string
	if f.Action != nil {
		a := string(*f.Action)
		action = &a
	}
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.EventPage{}, err
	}
	var page domain.EventPage
	err = r.db.WithTenantTx(ctx, f.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		rows, err := q.ListEvents(ctx, sqlcgen.ListEventsParams{
			Action: action, UserID: f.UserID, EntityID: f.EntityID,
			OccurredFrom: f.OccurredFrom, OccurredBefore: f.OccurredBefore,
			RowLimit: limit, RowOffset: offset,
		})
		if err != nil {
			return err
		}
		total, err := q.CountEvents(ctx, sqlcgen.CountEventsParams{
			Action: action, UserID: f.UserID, EntityID: f.EntityID,
			OccurredFrom: f.OccurredFrom, OccurredBefore: f.OccurredBefore,
		})
		if err != nil {
			return err
		}
		page.Total = int(total)
		page.Items, err = events(rows)
		return err
	})
	if err != nil {
		return domain.EventPage{}, fmt.Errorf("audit: list events: %w", err)
	}
	return page, nil
}

func events(rows []sqlcgen.ListEventsRow) ([]*domain.Event, error) {
	out := make([]*domain.Event, 0, len(rows))
	for _, row := range rows {
		var details map[string]string
		if err := json.Unmarshal(row.Details, &details); err != nil {
			return nil, fmt.Errorf("decode details of %s: %w", row.ID, err)
		}
		out = append(out, domain.RestoreEvent(domain.EventInput{
			ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, Role: row.Role,
			Action: domain.Action(row.Action), EntityID: row.EntityID, LocationID: row.LocationID,
			OccurredAt: row.OccurredAt.UTC(), Details: details,
		}))
	}
	return out, nil
}
