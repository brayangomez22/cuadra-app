package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// fileSummary is what reading a file found: its rows and products, and its
// errors (the first domain.MaxErrors, by row) with their total.
type fileSummary struct {
	Rows       int
	Products   int
	Errors     []domain.RowError
	ErrorCount int
}

func (f *fileSummary) addErrors(errs ...domain.RowError) {
	f.ErrorCount += len(errs)
	room := max(domain.MaxErrors-len(f.Errors), 0)
	f.Errors = append(f.Errors, errs[:min(room, len(errs))]...)
}

// sortErrors puts the errors in row order.
func (f *fileSummary) sortErrors() {
	slices.SortStableFunc(f.Errors, func(a, b domain.RowError) int { return cmp.Compare(a.Row, b.Row) })
}

// batchFunc receives the products of one batch and the row errors found
// while reading them.
type batchFunc func(groups []domain.Group, errs []domain.RowError) error

// fileReader reads a file one row at a time and hands its products to a
// batchFunc by batches of about batchRows rows, never splitting a product.
// Only the current batch is kept in memory.
type fileReader struct {
	layout    domain.Layout
	batchRows int
	onBatch   batchFunc
	summary   fileSummary
	pending   []domain.Group
	rows      int
	errs      []domain.RowError
}

// readBatches reads sheet with layout, calling onBatch for each batch. An
// error of onBatch stops the reading and is returned.
func (s *Service) readBatches(sheet Sheet, layout domain.Layout, onBatch batchFunc) (fileSummary, error) {
	r := &fileReader{layout: layout, batchRows: s.BatchRows, onBatch: onBatch}
	if err := r.read(sheet); err != nil {
		return fileSummary{}, err
	}
	return r.summary, nil
}

func (r *fileReader) read(sheet Sheet) error {
	var header *domain.Header
	grouper := domain.NewGrouper(r.layout.Variants)
	number := 0
	for cells, err := range sheet.Rows {
		if err != nil {
			return fmt.Errorf("%w: %w", domain.ErrUnreadableFile, err)
		}
		number++
		if header == nil {
			h, errs := domain.ParseHeader(r.layout, cells)
			if r.summary.addErrors(errs...); len(errs) > 0 {
				return nil
			}
			header = &h
			continue
		}
		if r.summary.Rows == domain.MaxRows {
			r.summary.addErrors(domain.RowError{Row: number, Code: domain.CodeTooManyRows})
			break
		}
		row, errs := header.Parse(number, cells, sheet.Style)
		if err := r.addRow(grouper, row, errs); err != nil {
			return err
		}
	}
	return r.finish(grouper)
}

func (r *fileReader) addRow(g *domain.Grouper, row domain.Row, errs []domain.RowError) error {
	if !row.Blank {
		r.summary.Rows++
	}
	done, ok, groupErrs := g.Add(row, len(errs) == 0)
	r.errs = append(append(r.errs, errs...), groupErrs...)
	if !ok {
		return nil
	}
	return r.addGroup(done)
}

func (r *fileReader) addGroup(g domain.Group) error {
	r.summary.Products++
	r.pending = append(r.pending, g)
	if r.rows += len(g.Variants); r.rows < r.batchRows {
		return nil
	}
	return r.flush()
}

func (r *fileReader) flush() error {
	if len(r.pending) == 0 && len(r.errs) == 0 {
		return nil
	}
	r.summary.addErrors(r.errs...)
	err := r.onBatch(r.pending, r.errs)
	r.pending, r.errs, r.rows = nil, nil, 0
	return err
}

// finish hands over the last product and batch; a file without data rows
// is reported in row 1.
func (r *fileReader) finish(g *domain.Grouper) error {
	if last, ok := g.Flush(); ok {
		if err := r.addGroup(last); err != nil {
			return err
		}
	}
	if err := r.flush(); err != nil {
		return err
	}
	if r.summary.Rows == 0 && r.summary.ErrorCount == 0 {
		r.summary.addErrors(domain.RowError{Row: 1, Code: domain.CodeEmptyFile})
	}
	return nil
}

// layout returns the tenant's file layout, with its variants module.
func (s *Service) layout(ctx context.Context, a Actor) (domain.Layout, error) {
	l, err := s.Catalog.Layout(ctx, a)
	if err != nil {
		return domain.Layout{}, err
	}
	l.Variants, err = s.Modules.IsActive(ctx, a.TenantID, variantsModule)
	return l, err
}

// problemError is the row error of a group the catalog rejected.
func problemError(g domain.Group, p *Problem) domain.RowError {
	return domain.RowError{Row: g.Row, Code: p.Code, Message: p.Message}
}
