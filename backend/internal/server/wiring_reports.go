package server

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// following builds the modules that follow how the business goes rather
// than run it: the reports, the users' comments and the getting-started
// checklist. It returns their routes.
func (w *wiring) following(sell sellingModules) ([]func(*http.ServeMux), error) {
	reportsModule, err := w.reports(sell.receivables)
	if err != nil {
		return nil, err
	}
	feedbackModule, err := w.feedback()
	if err != nil {
		return nil, err
	}
	onboardingModule, err := w.onboarding(sell.invoicing)
	if err != nil {
		return nil, err
	}
	return []func(*http.ServeMux){
		reportsModule.RegisterRoutes, feedbackModule.RegisterRoutes, onboardingModule.RegisterRoutes,
	}, nil
}

// reports builds the business reports, which export the receivables
// module's aging.
func (w *wiring) reports(r *receivables.Module) (*reports.Module, error) {
	return reports.New(reports.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Receivables:    ports.ReportsReceivables{Receivables: r},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}
