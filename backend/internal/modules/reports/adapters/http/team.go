package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// TeamUseCases are the use cases of what the team did (*app.Service).
type TeamUseCases interface {
	CashClosings(ctx context.Context, a app.Actor, q app.CashQuery) (domain.CashReport, error)
	TeamControl(ctx context.Context, a app.Actor, q app.TeamQuery) (domain.TeamReport, error)
}

type teamHandlers struct {
	uc TeamUseCases
	actors
}

// GetCashClosingsReport returns the period's cash closings by group.
func (h *teamHandlers) GetCashClosingsReport(ctx context.Context, req reportsapi.GetCashClosingsReportRequestObject) (reportsapi.GetCashClosingsReportResponseObject, error) {
	p := req.Params
	r, err := h.uc.CashClosings(ctx, h.actor(ctx), app.CashQuery{
		Range: days(p.From, p.To), LocationID: p.LocationId, Group: str(p.GroupBy),
	})
	if err != nil {
		return nil, err
	}
	items := make([]reportsapi.CashClosingsReportRow, 0, len(r.Rows))
	for _, row := range r.Rows {
		f := cashFigures(row.CashFigures)
		items = append(items, reportsapi.CashClosingsReportRow{
			Key: row.Key, Label: row.Label, Closings: f.Closings, WithDifference: f.WithDifference,
			SalesTotal: f.SalesTotal, Shortage: f.Shortage, Overage: f.Overage, Difference: f.Difference,
		})
	}
	return reportsapi.GetCashClosingsReport200JSONResponse{
		Period: period(r.Period), GroupBy: reportsapi.CashClosingsReportGroup(r.Group), Items: items, Totals: cashFigures(r.Totals),
	}, nil
}

func cashFigures(f domain.CashFigures) reportsapi.CashClosingFigures {
	return reportsapi.CashClosingFigures{
		Closings: f.Closings, WithDifference: f.WithDifference, SalesTotal: money(f.SalesTotal),
		Shortage: money(f.Shortage), Overage: money(f.Overage), Difference: money(f.Difference),
	}
}

// GetTeamControlReport returns what each user did, in total and by bucket.
func (h *teamHandlers) GetTeamControlReport(ctx context.Context, req reportsapi.GetTeamControlReportRequestObject) (reportsapi.GetTeamControlReportResponseObject, error) {
	p := req.Params
	r, err := h.uc.TeamControl(ctx, h.actor(ctx), app.TeamQuery{
		Range: days(p.From, p.To), LocationID: p.LocationId, Bucket: str(p.Bucket),
	})
	if err != nil {
		return nil, err
	}
	body := reportsapi.TeamControlReport{
		Period: period(r.Period), Bucket: reportsapi.ReportBucket(r.Bucket),
		Users: make([]reportsapi.TeamControlUser, 0, len(r.Users)), Series: make([]reportsapi.TeamControlPoint, 0, len(r.Series)),
	}
	for _, u := range r.Users {
		f := teamFigures(u.TeamFigures, r.Costs)
		body.Users = append(body.Users, reportsapi.TeamControlUser{
			UserId: u.UserID, Name: u.Name, Voids: f.Voids, VoidsTotal: f.VoidsTotal, Discounts: f.Discounts,
			DiscountsAmount: f.DiscountsAmount, DiscountRate: f.DiscountRate, PriceChanges: f.PriceChanges,
			Adjustments: f.Adjustments, AdjustmentsValue: f.AdjustmentsValue, CashDifferences: f.CashDifferences,
			CashShortage: f.CashShortage, CashOverage: f.CashOverage,
		})
	}
	for _, pt := range r.Series {
		f := teamFigures(pt.TeamFigures, r.Costs)
		body.Series = append(body.Series, reportsapi.TeamControlPoint{
			UserId: pt.UserID, BucketStart: date(pt.BucketStart), Voids: f.Voids, VoidsTotal: f.VoidsTotal,
			Discounts: f.Discounts, DiscountsAmount: f.DiscountsAmount, DiscountRate: f.DiscountRate,
			PriceChanges: f.PriceChanges, Adjustments: f.Adjustments, AdjustmentsValue: f.AdjustmentsValue,
			CashDifferences: f.CashDifferences, CashShortage: f.CashShortage, CashOverage: f.CashOverage,
		})
	}
	return reportsapi.GetTeamControlReport200JSONResponse(body), nil
}

func teamFigures(f domain.TeamFigures, costs bool) reportsapi.TeamControlFigures {
	return reportsapi.TeamControlFigures{
		Voids: f.Voids, VoidsTotal: money(f.VoidsTotal), Discounts: f.Discounts,
		DiscountsAmount: money(f.DiscountsAmount), DiscountRate: rate(f.DiscountRate()),
		PriceChanges: f.PriceChanges, Adjustments: f.Adjustments, AdjustmentsValue: optionalMoney(f.AdjustmentsValue, costs),
		CashDifferences: f.CashDifferences, CashShortage: money(f.CashShortage), CashOverage: money(f.CashOverage),
	}
}
