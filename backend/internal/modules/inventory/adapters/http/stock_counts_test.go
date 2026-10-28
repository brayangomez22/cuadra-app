package httpadapter_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// countStub answers the stock count use cases.
type countStub struct {
	startIn     app.StartCountInput
	entryIn     app.CountEntryInput
	entryFilter domain.EntryFilter
	lineFilter  app.LineFilter
	countFilter domain.CountFilter
	recount     []uuid.UUID
	manages     bool

	count   *domain.StockCount
	view    app.CountView
	entry   app.EntryView
	lines   app.CountLinesPage
	counts  domain.StockCountPage
	entries app.EntryPage
	err     error
}

func (s *countStub) StartStockCount(_ context.Context, _ app.Actor, in app.StartCountInput) (app.CountView, error) {
	s.startIn = in
	return s.view, s.err
}

func (s *countStub) GetStockCount(context.Context, app.Actor, uuid.UUID) (app.CountView, error) {
	return s.view, s.err
}

func (s *countStub) ListStockCounts(_ context.Context, _ app.Actor, f domain.CountFilter) (domain.StockCountPage, error) {
	s.countFilter = f
	return s.counts, s.err
}

func (s *countStub) StockCountLines(_ context.Context, _ app.Actor, _ uuid.UUID, f app.LineFilter) (app.CountLinesPage, error) {
	s.lineFilter = f
	return s.lines, s.err
}

func (s *countStub) RequestRecount(_ context.Context, _ app.Actor, _ uuid.UUID, variantIDs []uuid.UUID) error {
	s.recount = variantIDs
	return s.err
}

func (s *countStub) ApproveStockCount(context.Context, app.Actor, uuid.UUID) (*domain.StockCount, error) {
	return s.count, s.err
}

func (s *countStub) CancelStockCount(context.Context, app.Actor, uuid.UUID) (*domain.StockCount, error) {
	return s.count, s.err
}

func (s *countStub) RecordCountEntry(_ context.Context, _ app.Actor, in app.CountEntryInput) (app.EntryView, error) {
	s.entryIn = in
	return s.entry, s.err
}

func (s *countStub) DeleteCountEntry(_ context.Context, _ app.Actor, _, _ uuid.UUID, manages bool) error {
	s.manages = manages
	return s.err
}

func (s *countStub) ListCountEntries(_ context.Context, _ app.Actor, f domain.EntryFilter, manages bool) (app.EntryPage, error) {
	s.entryFilter, s.manages = f, manages
	return s.entries, s.err
}

func anOpenCount(t *testing.T) *domain.StockCount {
	t.Helper()
	c, err := domain.NewStockCount(domain.StockCountSpec{
		ID: newID(), TenantID: owner.TenantID, LocationID: newID(), Mode: domain.CountSum,
		StartedBy: owner.UserID, StartedAt: now,
	})
	require.NoError(t, err)
	return c
}

func TestStartStockCountHTTP(t *testing.T) {
	t.Run("inicia un conteo total en modo suma por defecto", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.view = app.CountView{StockCount: anOpenCount(t), Progress: domain.CountProgress{Lines: 120}}
		location := newID()
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts", `{"location_id":"`+location.String()+`"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.StartCountInput{LocationID: location, Mode: domain.CountSum}, uc.startIn)
		body := decodeMap(t, rec)
		require.Equal(t, "open", body["status"])
		require.EqualValues(t, 120, body["lines"])
		require.EqualValues(t, 0, body["counted_lines"])
		require.Equal(t, []any{}, body["category_ids"])
		require.NotContains(t, body, "summary")
		require.NotContains(t, body, "closed_at")
	})

	t.Run("inicia un conteo parcial en modo revisión", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.view = app.CountView{StockCount: anOpenCount(t)}
		category := newID()
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts",
			`{"location_id":"`+newID().String()+`","category_ids":["`+category.String()+`"],"mode":"review"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, []uuid.UUID{category}, uc.startIn.CategoryIDs)
		require.Equal(t, domain.CountReview, uc.startIn.Mode)
	})

	errorCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"otra sesión abierta en la sede es un conflicto", domain.ErrCountInProgress, http.StatusConflict, "stock_count_in_progress"},
		{"una categoría que no existe es un dato inválido", domain.ErrCategoryNotFound, http.StatusUnprocessableEntity, "category_not_found"},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			uc := &stubUseCases{}
			uc.countStub.err = tc.err
			rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts", `{"location_id":"`+newID().String()+`"}`)
			requireError(t, rec, tc.status, tc.code)
		})
	}
}

func TestCountEntriesHTTP(t *testing.T) {
	t.Run("quien cuenta no ve el stock esperado", func(t *testing.T) {
		uc := &stubUseCases{}
		c := anOpenCount(t)
		e, err := c.NewEntry(newID(), newID(), dec("12.5"), owner.UserID, now)
		require.NoError(t, err)
		uc.entry = app.EntryView{CountEntry: e, ProductID: newID()}
		uc.view = app.CountView{StockCount: c, Progress: domain.CountProgress{Lines: 3, Counted: 1}}
		p := warehouse()

		rec := serve(t, uc, &p, http.MethodPost, "/api/v1/stock-counts/"+c.ID().String()+"/entries",
			`{"variant_id":"`+e.VariantID().String()+`","quantity":"12.5"}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "12.5", decodeMap(t, rec)["quantity"])
		require.True(t, uc.entryIn.Quantity.Equal(dec("12.5")))

		got := serve(t, uc, &p, http.MethodGet, "/api/v1/stock-counts/"+c.ID().String(), "")
		require.Equal(t, http.StatusOK, got.Code)
		for _, body := range []string{rec.Body.String(), got.Body.String()} {
			require.NotContains(t, body, "expected")
		}
	})

	t.Run("quien no administra solo pide sus conteos", func(t *testing.T) {
		uc := &stubUseCases{}
		p := warehouse()
		rec := serve(t, uc, &p, http.MethodGet, "/api/v1/stock-counts/"+newID().String()+"/entries", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.False(t, uc.manages)

		serve(t, uc, &owner, http.MethodGet, "/api/v1/stock-counts/"+newID().String()+"/entries", "")
		require.True(t, uc.manages)
	})

	t.Run("borrar lo contado por otro es prohibido", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.countStub.err = domain.ErrCountEntryNotYours
		p := warehouse()
		rec := serve(t, uc, &p, http.MethodDelete,
			"/api/v1/stock-counts/"+newID().String()+"/entries/"+newID().String(), "")
		requireError(t, rec, http.StatusForbidden, "count_entry_not_yours")
	})

	t.Run("contar en un conteo cerrado es un conflicto", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.countStub.err = domain.ErrCountNotOpen
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts/"+newID().String()+"/entries",
			`{"variant_id":"`+newID().String()+`","quantity":"1"}`)
		requireError(t, rec, http.StatusConflict, "stock_count_not_open")
	})

	t.Run("un conteo que no existe responde 404", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.countStub.err = domain.ErrCountNotFound
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/stock-counts/"+newID().String(), "")
		requireError(t, rec, http.StatusNotFound, "stock_count_not_found")
	})
}
