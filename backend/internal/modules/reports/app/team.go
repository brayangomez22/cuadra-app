package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CashQuery asks for the cash closings report.
type CashQuery struct {
	Range
	LocationID *uuid.UUID
	Group      string
}

// CashClosings returns the period's cash closings by group, with totals.
func (s *Service) CashClosings(ctx context.Context, a Actor, q CashQuery) (_ domain.CashReport, err error) {
	ctx, span := s.startSpan(ctx, "CashClosings", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "cash_closings", time.Now())

	period, err := q.period()
	if err != nil {
		return domain.CashReport{}, err
	}
	group, err := domain.ParseCashGroup(q.Group)
	if err != nil {
		return domain.CashReport{}, err
	}
	closings, err := s.Team.Closings(ctx, domain.CashFilter{Scope: scope(a, q.LocationID), Period: period})
	if err != nil {
		return domain.CashReport{}, err
	}
	r := domain.NewCashReport(closings, group)
	r.Period = period
	return r, nil
}

// TeamQuery asks for the team control report.
type TeamQuery struct {
	Range
	LocationID *uuid.UUID
	Bucket     string
}

// TeamControl returns what each user did in the period that the owner
// watches, in total and as a trend by bucket.
func (s *Service) TeamControl(ctx context.Context, a Actor, q TeamQuery) (_ domain.TeamReport, err error) {
	ctx, span := s.startSpan(ctx, "TeamControl", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "team_control", time.Now())

	period, err := q.period()
	if err != nil {
		return domain.TeamReport{}, err
	}
	bucket, err := domain.ParseBucket(q.Bucket)
	if err != nil {
		return domain.TeamReport{}, err
	}
	points, err := s.Team.TeamControl(ctx, domain.TeamFilter{Scope: scope(a, q.LocationID), Period: period, Bucket: bucket})
	if err != nil {
		return domain.TeamReport{}, err
	}
	r := domain.TeamReport{Period: period, Bucket: bucket, Users: domain.SumByUser(points), Series: points, Costs: true}
	if !a.SeesCosts {
		r.HideCosts()
	}
	return r, nil
}
