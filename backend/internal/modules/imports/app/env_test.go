package app_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

var testNow = time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc      *app.Service
	imports  *fakeImports
	catalog  *fakeCatalog
	stock    *fakeStock
	modules  *fakeModules
	actor    app.Actor
	location uuid.UUID
	spans    *tracetest.SpanRecorder
	metrics  *sdkmetric.ManualReader
}

// testBatchRows keeps the batches small so a few rows make several.
const testBatchRows = 2

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		imports:  newFakeImports(),
		catalog:  newFakeCatalog(),
		stock:    &fakeStock{locations: map[uuid.UUID]bool{}},
		modules:  &fakeModules{active: map[string]bool{"variants": true}},
		actor:    app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())},
		location: uuid.Must(uuid.NewV7()),
		spans:    tracetest.NewSpanRecorder(),
		metrics:  sdkmetric.NewManualReader(),
	}
	e.stock.locations[e.location] = true
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Imports:        e.imports,
		Tx:             fakeTx{catalog: e.catalog, stock: e.stock, imports: e.imports},
		Catalog:        e.catalog,
		Stock:          e.stock,
		Modules:        e.modules,
		Sheets:         fakeSheets{},
		BatchRows:      testBatchRows,
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// file is the content of an import file for fakeSheets: one line per row,
// cells separated by "|", with the header of the template.
func file(rows ...string) []byte {
	header := strings.Join(domain.Layout{Variants: true, Axes: testAxes}.Headers(), "|")
	return []byte(strings.Join(append([]string{header}, rows...), "\n"))
}

// line is a data row of file: product, category, unit, SKU, cost, price,
// initial stock and the size axis; the other columns stay empty.
func line(product, sku, price, stock, size string) string {
	values := map[string]string{
		domain.ColumnProduct: product, domain.ColumnUnit: "und", domain.ColumnSKU: sku,
		domain.ColumnCost: "100", domain.ColumnPrice: price, domain.ColumnStock: stock,
		domain.AxisPrefix + "talla": size,
	}
	headers := domain.Layout{Variants: true, Axes: testAxes}.Headers()
	cells := make([]string, len(headers))
	for i, h := range headers {
		cells[i] = values[h]
	}
	return strings.Join(cells, "|")
}

// upload uploads data as a CSV file with the update policy.
func (e *env) upload(t *testing.T, data []byte) app.Preview {
	t.Helper()
	p, err := e.svc.UploadImport(t.Context(), e.actor, app.UploadInput{
		FileName: "productos.csv", Format: domain.FormatCSV, Data: data,
		LocationID: e.location, Policy: domain.UpdateExisting,
	})
	require.NoError(t, err)
	return p
}

// confirmed uploads data and confirms it.
func (e *env) confirmed(t *testing.T, data []byte) *domain.Import {
	t.Helper()
	p := e.upload(t, data)
	require.Equal(t, domain.StatusValidated, p.Import.Status(), "errors: %v", p.Errors)
	imp, err := e.svc.ConfirmImport(t.Context(), e.actor, p.Import.ID())
	require.NoError(t, err)
	return imp
}

// productRows returns n rows of n products of one variant each.
func productRows(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = line(fmt.Sprintf("Producto %d", i), fmt.Sprintf("SKU-%d", i), "1000", "5", "")
	}
	return rows
}

// span returns the attributes of the only ended span named name.
func (e *env) span(t *testing.T, name string) map[attribute.Key]attribute.Value {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range found[0].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}

// counter returns the sum of an int64 counter's data points whose attribute
// key equals value (any data point when key is "").
func (e *env) counter(t *testing.T, name, key, value string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			sum, ok := m.Data.(metricdata.Sum[int64])
			if m.Name != name || !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value(attribute.Key(key)); key == "" || ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}
