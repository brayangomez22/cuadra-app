package httpadapter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// contentTypes are the media types of the export formats.
var contentTypes = map[sheet.Format]string{
	sheet.XLSX: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	sheet.CSV:  "text/csv; charset=utf-8",
}

// exporter reads a report with the export's filters and builds its table.
type exporter func(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error)

// exporters are the tables of each report.
var exporters = map[reportsapi.ReportKind]exporter{
	reportsapi.Sales:            exportSales,
	reportsapi.Products:         exportProducts,
	reportsapi.SaleProfits:      exportSaleProfits,
	reportsapi.IdleStock:        exportIdleStock,
	reportsapi.CashClosings:     exportCashClosings,
	reportsapi.TeamControl:      exportTeamControl,
	reportsapi.StockAdjustments: exportAdjustments,
	reportsapi.Receivables:      exportReceivables,
}

type exportHandler struct {
	uc UseCases
	actors
}

// ExportReport returns a report as an .xlsx or a CSV download.
func (h *exportHandler) ExportReport(ctx context.Context, req reportsapi.ExportReportRequestObject) (reportsapi.ExportReportResponseObject, error) {
	p := req.Params
	export, ok := exporters[p.Report]
	if !ok {
		return nil, domain.ErrInvalidFilter
	}
	table, err := export(ctx, h.uc, h.actor(ctx), p)
	if err != nil {
		return nil, err
	}
	format := sheet.XLSX
	if p.Format != nil {
		format = sheet.Format(*p.Format)
	}
	data, err := sheet.Write(table, format)
	if err != nil {
		return nil, err
	}
	h.uc.Exported(ctx, string(p.Report), string(format))
	return download{name: fileName(p), format: format, data: data}, nil
}

// exportRange is the period of a report that needs one: both days, or
// ErrInvalidFilter.
func exportRange(p reportsapi.ExportReportParams) (app.Range, error) {
	if p.From == nil || p.To == nil {
		return app.Range{}, domain.ErrInvalidFilter
	}
	return app.Range{From: day(*p.From), To: day(*p.To)}, nil
}

// fileName names the file by the report and its period.
func fileName(p reportsapi.ExportReportParams) string {
	name := "reporte-" + string(p.Report)
	if p.From != nil && p.To != nil {
		name += "-" + day(*p.From).String() + "-a-" + day(*p.To).String()
	}
	return name
}

// download writes the file with a name, so the browser saves it.
type download struct {
	name   string
	format sheet.Format
	data   []byte
}

func (d download) VisitExportReportResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", contentTypes[d.format])
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, d.name, d.format))
	w.Header().Set("Content-Length", strconv.Itoa(len(d.data)))
	w.WriteHeader(http.StatusOK)
	_, err := bytes.NewReader(d.data).WriteTo(w)
	return err
}
