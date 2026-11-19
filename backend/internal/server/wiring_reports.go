package server

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

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
