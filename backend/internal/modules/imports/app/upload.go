package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// How much of the file a preview shows.
const (
	previewErrors   = 50
	previewProducts = 10
)

// UploadInput is an uploaded file and what to do with it.
type UploadInput struct {
	FileName   string
	Format     domain.Format
	Data       []byte
	LocationID uuid.UUID
	Policy     domain.ExistingPolicy
}

// Preview is a validated (or rejected) import: its first errors and its
// first products, as the file describes them.
type Preview struct {
	Import *domain.Import
	Errors []domain.RowError
	Sample []domain.Group
}

// UploadImport reads and validates a file and stores it with its errors:
// each row against the tenant's columns, and each product against the
// catalog (without writing to it). The import is validated only if no row
// has an error, so nothing is written from a file that has one.
func (s *Service) UploadImport(ctx context.Context, a Actor, in UploadInput) (_ Preview, err error) {
	ctx, span := s.startSpan(ctx, "UploadImport", a, attribute.String("import.format", string(in.Format)))
	defer func() { usecase.End(span, err) }()

	if len(in.Data) > domain.MaxFileBytes {
		return Preview{}, domain.ErrFileTooLarge
	}
	if err := s.Stock.RequireLocation(ctx, a, in.LocationID); err != nil {
		return Preview{}, err
	}
	summary, sample, err := s.validate(ctx, a, in)
	if err != nil {
		return Preview{}, err
	}
	imp, err := domain.NewImport(domain.NewImportParams{
		TenantID: a.TenantID, UserID: a.UserID, LocationID: in.LocationID,
		FileName: in.FileName, Format: in.Format, Policy: in.Policy,
		Rows: summary.Rows, Products: summary.Products, Errors: summary.ErrorCount,
	}, s.Now())
	if err != nil {
		return Preview{}, err
	}
	span.SetAttributes(importAttr(imp.ID()))
	if err := s.Imports.Create(ctx, imp, in.Data, summary.Errors); err != nil {
		return Preview{}, err
	}
	s.uploads.Add(ctx, 1, resultAttr(string(imp.Status())))
	return Preview{Import: imp, Errors: summary.Errors[:min(previewErrors, len(summary.Errors))], Sample: sample}, nil
}

// validate reads the file and checks its products with the catalog by
// batches; it returns what it found and the first products.
func (s *Service) validate(ctx context.Context, a Actor, in UploadInput) (fileSummary, []domain.Group, error) {
	layout, err := s.layout(ctx, a)
	if err != nil {
		return fileSummary{}, nil, err
	}
	sheet, err := s.Sheets.Rows(in.Format, in.Data)
	if err != nil {
		return fileSummary{}, nil, err
	}
	var sample []domain.Group
	var problems []domain.RowError
	summary, err := s.readBatches(sheet, layout, func(groups []domain.Group, _ []domain.RowError) error {
		sample = append(sample, groups[:min(previewProducts-len(sample), len(groups))]...)
		found, err := s.check(ctx, a, groups, in.Policy)
		problems = append(problems, found...)
		return err
	})
	if err != nil {
		return fileSummary{}, nil, err
	}
	summary.addErrors(problems...)
	summary.sortErrors()
	return summary, sample, nil
}

// check asks the catalog about the valid groups and returns the problems
// it finds.
func (s *Service) check(ctx context.Context, a Actor, groups []domain.Group, p domain.ExistingPolicy) ([]domain.RowError, error) {
	valid := make([]domain.Group, 0, len(groups))
	for _, g := range groups {
		if !g.Invalid {
			valid = append(valid, g)
		}
	}
	if len(valid) == 0 {
		return nil, nil
	}
	results, err := s.Catalog.CheckProducts(ctx, a, valid, p)
	if err != nil {
		return nil, err
	}
	var problems []domain.RowError
	for i, r := range results {
		if r.Problem != nil {
			problems = append(problems, problemError(valid[i], r.Problem))
		}
	}
	return problems, nil
}
