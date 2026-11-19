package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// TeamRepository implements domain.TeamReader.
type TeamRepository struct{ db *db.DB }

// NewTeamRepository returns a TeamRepository on d.
func NewTeamRepository(d *db.DB) *TeamRepository { return &TeamRepository{db: d} }

// Closings reads the cash sessions closed in the period.
func (r *TeamRepository) Closings(ctx context.Context, f domain.CashFilter) ([]domain.Closing, error) {
	arg := sqlcgen.CashClosingsParams{PeriodStart: f.Period.Start(), PeriodEnd: f.Period.End(), LocationID: f.LocationID}
	closings, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) ([]domain.Closing, error) {
		rows, err := sqlcgen.New(tx).CashClosings(ctx, arg)
		if err != nil {
			return nil, err
		}
		out := make([]domain.Closing, 0, len(rows))
		for _, x := range rows {
			d, err := day(x.ClosedDay)
			if err != nil {
				return nil, err
			}
			out = append(out, domain.Closing{
				UserID: x.UserID, UserName: x.UserName, LocationID: x.LocationID, LocationName: x.LocationName,
				Day: calendar.DateOf(d), SalesTotal: x.SalesTotal, Difference: x.Difference,
			})
		}
		return out, nil
	})
	if err != nil {
		return nil, fmt.Errorf("reports: read cash closings: %w", err)
	}
	return closings, nil
}

// TeamControl reads each user's figures in each bucket with any.
func (r *TeamRepository) TeamControl(ctx context.Context, f domain.TeamFilter) ([]domain.TeamPoint, error) {
	arg := sqlcgen.TeamControlParams{
		Bucket: string(f.Bucket), PeriodStart: f.Period.Start(), PeriodEnd: f.Period.End(), LocationID: f.LocationID,
	}
	points, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) ([]domain.TeamPoint, error) {
		return collect(teamPoint)(sqlcgen.New(tx).TeamControl(ctx, arg))
	})
	if err != nil {
		return nil, fmt.Errorf("reports: read team control: %w", err)
	}
	return points, nil
}

func teamPoint(x sqlcgen.TeamControlRow) domain.TeamPoint {
	return domain.TeamPoint{
		UserID: x.UserID, Name: x.Name, BucketStart: calendar.DateOf(x.BucketStart.Time),
		TeamFigures: domain.TeamFigures{
			Voids: int(x.Voids), VoidsTotal: x.VoidsTotal, Discounts: int(x.Discounts),
			DiscountsAmount: x.DiscountsAmount, DiscountsGross: x.DiscountsGross, PriceChanges: int(x.PriceChanges),
			Adjustments: int(x.Adjustments), AdjustmentsValue: x.AdjustmentsValue,
			CashDifferences: int(x.CashDifferences), CashShortage: x.CashShortage, CashOverage: x.CashOverage,
		},
	}
}
