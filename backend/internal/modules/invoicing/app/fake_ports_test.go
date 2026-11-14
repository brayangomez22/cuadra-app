package app_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// fakeInvoicer is the provider: it answers with result, or fails with err,
// and remembers what it received by number.
type fakeInvoicer struct {
	result   app.Result
	err      error
	issued   []app.IssueRequest
	received map[app.DocumentRef]app.Result
	lookups  int
	// lookupErr fails the look ups.
	lookupErr error
}

func newInvoicer() *fakeInvoicer {
	return &fakeInvoicer{
		result:   app.Result{Status: domain.StatusAccepted, ProviderID: "prov-1", Code: "cude-1", QR: "https://qr"},
		received: map[app.DocumentRef]app.Result{},
	}
}

func (f *fakeInvoicer) Issue(_ context.Context, req app.IssueRequest) (app.Result, error) {
	if f.err != nil {
		return app.Result{}, f.err
	}
	f.issued = append(f.issued, req)
	f.received[req.DocumentRef] = f.result
	return f.result, nil
}

func (f *fakeInvoicer) Lookup(_ context.Context, ref app.DocumentRef) (app.Result, bool, error) {
	f.lookups++
	if f.lookupErr != nil {
		return app.Result{}, false, f.lookupErr
	}
	r, ok := f.received[ref]
	return r, ok, nil
}

// down makes the provider fail with a temporary error.
func (f *fakeInvoicer) down() {
	f.err = &app.ProviderError{Reason: app.ReasonServer, Err: errors.New("503 Service Unavailable")}
}

type fakeSales struct{ sales map[uuid.UUID]app.SaleData }

func (f fakeSales) Sale(_ context.Context, _, _, saleID uuid.UUID) (app.SaleData, error) {
	s, ok := f.sales[saleID]
	if !ok {
		return app.SaleData{}, errors.New("sale not found")
	}
	return s, nil
}

type fakeBuyers struct{ buyers map[uuid.UUID]domain.Buyer }

func (f fakeBuyers) Buyer(_ context.Context, _, _, customerID uuid.UUID) (domain.Buyer, error) {
	b, ok := f.buyers[customerID]
	if !ok {
		return domain.Buyer{}, errors.New("customer not found")
	}
	return b, nil
}
