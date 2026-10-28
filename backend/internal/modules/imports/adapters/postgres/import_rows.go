package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// errCountOutOfRange reports a count that does not fit the int4 columns.
var errCountOutOfRange = errors.New("imports: count out of range")

func save(ctx context.Context, q *sqlcgen.Queries, imp *domain.Import) error {
	c := imp.Counts()
	n, err := int32s(c.Created, c.Updated, c.Skipped, c.Errors, imp.BatchesDone())
	if err != nil {
		return err
	}
	affected, err := q.SaveImport(ctx, sqlcgen.SaveImportParams{
		TenantID: imp.TenantID(), ID: imp.ID(), Status: string(imp.Status()),
		Created: n[0], Updated: n[1], Skipped: n[2], Errors: n[3], BatchesDone: n[4],
		StartedAt: imp.StartedAt(), FinishedAt: imp.FinishedAt(),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrImportNotFound
	}
	return nil
}

// addErrors stores errs, up to domain.MaxErrors per call.
func addErrors(ctx context.Context, q *sqlcgen.Queries, imp *domain.Import, errs []domain.RowError) error {
	errs = errs[:min(len(errs), domain.MaxErrors)]
	if len(errs) == 0 {
		return nil
	}
	p := sqlcgen.AddImportErrorsParams{TenantID: imp.TenantID(), ImportID: imp.ID()}
	for _, e := range errs {
		row, err := int32s(e.Row)
		if err != nil {
			return err
		}
		p.RowNumbers = append(p.RowNumbers, row[0])
		p.ColumnNames = append(p.ColumnNames, e.Column)
		p.Codes = append(p.Codes, e.Code)
		p.Messages = append(p.Messages, e.Message)
	}
	return q.AddImportErrors(ctx, p)
}

func importFromRow(row sqlcgen.GetImportRow) (*domain.Import, error) {
	return domain.RehydrateImport(domain.ImportSnapshot{
		ID: row.ID, TenantID: row.TenantID, CreatedBy: row.CreatedBy, LocationID: row.LocationID,
		FileName: row.FileName, Format: domain.Format(row.Format), Policy: domain.ExistingPolicy(row.Policy),
		Status: domain.Status(row.Status),
		Counts: domain.Counts{
			Rows: int(row.TotalRows), Products: int(row.TotalProducts), Created: int(row.Created),
			Updated: int(row.Updated), Skipped: int(row.Skipped), Errors: int(row.Errors),
		},
		BatchesDone: int(row.BatchesDone), CreatedAt: row.CreatedAt.UTC(),
		StartedAt: utc(row.StartedAt), FinishedAt: utc(row.FinishedAt),
	})
}

// utc returns t in UTC, nil for nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// int32s converts counts to the columns' int4.
func int32s(counts ...int) ([]int32, error) {
	out := make([]int32, 0, len(counts))
	for _, n := range counts {
		if n < 0 || n > math.MaxInt32 {
			return nil, errCountOutOfRange
		}
		out = append(out, int32(n))
	}
	return out, nil
}
