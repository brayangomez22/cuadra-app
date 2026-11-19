package app_test

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// fakeReaders answers every report with what the test set, and keeps the
// last filter of each, so a test sees what the use case asked for. err, if
// set, is what every read fails with.
type fakeReaders struct {
	err error

	sales       domain.SalesReport
	sold        []domain.SoldVariant
	saleProfits domain.SaleProfitsReport
	idle        []domain.IdleLine
	adjustments domain.AdjustmentsReport
	closings    []domain.Closing
	points      []domain.TeamPoint

	salesFilter      domain.SalesFilter
	productFilter    domain.ProductFilter
	saleProfitFilter domain.SaleProfitFilter
	idleFilter       domain.IdleFilter
	adjustmentFilter domain.AdjustmentFilter
	cashFilter       domain.CashFilter
	teamFilter       domain.TeamFilter
	reads            int
}

func (f *fakeReaders) Sales(_ context.Context, q domain.SalesFilter) (domain.SalesReport, error) {
	f.salesFilter, f.reads = q, f.reads+1
	return f.sales, f.err
}

func (f *fakeReaders) SoldVariants(_ context.Context, q domain.ProductFilter) ([]domain.SoldVariant, error) {
	f.productFilter, f.reads = q, f.reads+1
	return f.sold, f.err
}

func (f *fakeReaders) SaleProfits(_ context.Context, q domain.SaleProfitFilter) (domain.SaleProfitsReport, error) {
	f.saleProfitFilter, f.reads = q, f.reads+1
	return f.saleProfits, f.err
}

func (f *fakeReaders) IdleVariants(_ context.Context, q domain.IdleFilter) ([]domain.IdleLine, error) {
	f.idleFilter, f.reads = q, f.reads+1
	return f.idle, f.err
}

func (f *fakeReaders) Adjustments(_ context.Context, q domain.AdjustmentFilter) (domain.AdjustmentsReport, error) {
	f.adjustmentFilter, f.reads = q, f.reads+1
	return f.adjustments, f.err
}

func (f *fakeReaders) Closings(_ context.Context, q domain.CashFilter) ([]domain.Closing, error) {
	f.cashFilter, f.reads = q, f.reads+1
	return f.closings, f.err
}

func (f *fakeReaders) TeamControl(_ context.Context, q domain.TeamFilter) ([]domain.TeamPoint, error) {
	f.teamFilter, f.reads = q, f.reads+1
	return f.points, f.err
}

// fakeAging is the receivables port: it returns report and keeps the sort
// and the tenant it was asked for.
type fakeAging struct {
	report   domain.AgingExport
	sort     string
	tenantID uuid.UUID
}

func (f *fakeAging) Aging(_ context.Context, tenantID, _ uuid.UUID, sort string) (domain.AgingExport, error) {
	f.tenantID, f.sort = tenantID, sort
	return f.report, nil
}
