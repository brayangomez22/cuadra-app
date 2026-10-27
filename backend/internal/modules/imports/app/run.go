package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// errBatchRejected rolls back a batch the catalog rejected; the import
// fails with the batch's problems.
var errBatchRejected = errors.New("imports: batch rejected")

// RunImport writes a queued import, batch by batch: each batch, its stock
// entries and the checkpoint (BatchesDone) are one transaction. A retried
// job reads the file again and skips the batches already written, so no
// product or stock entry is written twice. A batch the catalog rejects is
// rolled back and the import fails with its errors (the earlier batches
// stay; uploading the file again with the update policy goes on from
// there). An infrastructure error is returned, and the job retries.
func (s *Service) RunImport(ctx context.Context, tenantID, id uuid.UUID) (err error) {
	ctx, span := s.spans.Start(ctx, "RunImport", attribute.String("tenant.id", tenantID.String()), importAttr(id))
	defer func() { usecase.End(span, err) }()

	imp, err := s.Imports.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	span.SetAttributes(attribute.String("user.id", imp.CreatedBy().String()))
	switch err := imp.Start(s.Now()); {
	case errors.Is(err, domain.ErrImportFinished):
		return nil
	case errors.Is(err, domain.ErrImportNotQueued):
		s.Logger.WarnContext(ctx, "import job for an unconfirmed import", slog.String("import_id", id.String()))
		return nil
	case err != nil:
		return err
	}
	if err := s.Imports.Save(ctx, imp, nil); err != nil {
		return err
	}
	failure, err := s.writeFile(ctx, imp)
	if err != nil {
		return err
	}
	return s.finish(ctx, imp, failure)
}

// writeFile writes the batches after the checkpoint; failure are the
// errors of the batch that could not be written, if any.
func (s *Service) writeFile(ctx context.Context, imp *domain.Import) ([]domain.RowError, error) {
	a := Actor{TenantID: imp.TenantID(), UserID: imp.CreatedBy()}
	data, err := s.Imports.File(ctx, a.TenantID, imp.ID())
	if err != nil {
		return nil, err
	}
	layout, err := s.layout(ctx, a)
	if err != nil {
		return nil, err
	}
	sheet, err := s.Sheets.Rows(imp.Format(), data)
	if err != nil {
		return nil, err
	}
	var failure []domain.RowError
	batch := 0
	_, err = s.readBatches(sheet, layout, func(groups []domain.Group, errs []domain.RowError) error {
		batch++
		switch {
		case batch <= imp.BatchesDone():
			return nil
		case len(errs) > 0: // the tenant's configuration changed after the upload
			failure = errs
			return errBatchRejected
		}
		rejected, err := s.writeBatch(ctx, a, imp, groups)
		failure = rejected
		return err
	})
	if errors.Is(err, errBatchRejected) {
		return failure, nil
	}
	return nil, err
}

// writeBatch writes one batch with its initial stock and moves the
// checkpoint, in one transaction. A rejected batch is rolled back and its
// problem returned.
func (s *Service) writeBatch(ctx context.Context, a Actor, imp *domain.Import, groups []domain.Group) ([]domain.RowError, error) {
	var problem *domain.RowError
	var tally domain.BatchResult
	err := s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		results, err := s.Catalog.ImportProducts(ctx, a, groups, imp.Policy())
		if err != nil {
			return err
		}
		if last := len(results) - 1; last >= 0 && results[last].Problem != nil {
			p := problemError(groups[last], results[last].Problem)
			problem = &p
			return errBatchRejected
		}
		if err := s.receive(ctx, a, imp, groups, results); err != nil {
			return err
		}
		tally = tallyResults(results)
		if err := imp.RecordBatch(tally); err != nil {
			return err
		}
		return s.Imports.Save(ctx, imp, nil)
	})
	if problem != nil {
		return []domain.RowError{*problem}, errBatchRejected
	}
	if err != nil {
		return nil, err
	}
	s.countProducts(ctx, tally)
	return nil, nil
}

// receive registers the initial stock of the variants the batch created.
func (s *Service) receive(ctx context.Context, a Actor, imp *domain.Import, groups []domain.Group, results []ProductResult) error {
	var entries []StockEntry
	for i, r := range results {
		for j, v := range r.Variants {
			line := groups[i].Variants[j]
			if v.Created && line.HasStock && line.Stock.IsPositive() {
				entries = append(entries, StockEntry{VariantID: v.ID, Quantity: line.Stock, UnitCost: line.Cost})
			}
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return s.Stock.ReceiveInitial(ctx, a, imp.LocationID(), entries)
}

func tallyResults(results []ProductResult) domain.BatchResult {
	var t domain.BatchResult
	for _, r := range results {
		switch r.Outcome {
		case OutcomeCreated:
			t.Created++
		case OutcomeUpdated:
			t.Updated++
		case OutcomeSkipped:
			t.Skipped++
		}
	}
	return t
}

func (s *Service) countProducts(ctx context.Context, t domain.BatchResult) {
	for result, n := range map[Outcome]int{OutcomeCreated: t.Created, OutcomeUpdated: t.Updated, OutcomeSkipped: t.Skipped} {
		if n > 0 {
			s.products.Add(ctx, int64(n), resultAttr(string(result)))
		}
	}
}

// finish ends the import: failed with failure's errors, or completed.
func (s *Service) finish(ctx context.Context, imp *domain.Import, failure []domain.RowError) error {
	now := s.Now()
	if len(failure) > 0 {
		if err := imp.Fail(now, len(failure)); err != nil {
			return err
		}
	} else if err := imp.Complete(now); err != nil {
		return err
	}
	if err := s.Imports.Save(ctx, imp, failure); err != nil {
		return err
	}
	s.runs.Add(ctx, 1, resultAttr(string(imp.Status())))
	if started := imp.StartedAt(); started != nil {
		s.duration.Record(ctx, now.Sub(*started).Seconds())
	}
	return nil
}
