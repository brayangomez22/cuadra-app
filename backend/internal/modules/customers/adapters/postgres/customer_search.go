package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Search returns one page of the tenant's customers matching f, and how many
// match in total, read in the same transaction.
func (r *CustomerRepository) Search(ctx context.Context, f domain.CustomerFilter) ([]*domain.Customer, int, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	params := sqlcgen.SearchCustomersParams{
		Active: activeFilter(f.Status), SegmentID: f.SegmentID, Terms: strings.Fields(f.Text),
		Document: documentText(f.Text), Text: f.Text, RowLimit: limit, RowOffset: offset,
	}
	var rows []sqlcgen.SearchCustomersRow
	var total int64
	err = r.db.WithTenantTx(ctx, f.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.SearchCustomers(ctx, params); err != nil {
			return err
		}
		total, err = q.CountCustomers(ctx, sqlcgen.CountCustomersParams{
			Active: params.Active, SegmentID: params.SegmentID, Terms: params.Terms, Document: params.Document,
		})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("customers: search customers: %w", err)
	}
	out := make([]*domain.Customer, 0, len(rows))
	for _, row := range rows {
		c, err := customerFromRow(customerRow(row))
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, int(total), nil
}

// activeFilter is the active column a status asks for; nil for all.
func activeFilter(s domain.CustomerStatus) *bool {
	switch s {
	case domain.StatusActive:
		return new(true)
	case domain.StatusInactive:
		return new(false)
	default:
		return nil
	}
}

// documentText is text as a document number is stored, without the
// separators people type ("1.020.304.050", "900123456-8"), or "" if text
// cannot be one.
func documentText(text string) string {
	doc := strings.ToUpper(strings.NewReplacer(".", "", " ", "", "-", "").Replace(text))
	if doc == "" || strings.ContainsFunc(doc, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'A' || r > 'Z')
	}) {
		return ""
	}
	return doc
}
