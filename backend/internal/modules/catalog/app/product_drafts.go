package app

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ProductDraft is a product as an import file describes it: its category
// by name, its unit by code and its attributes by key. Variants[0] is the
// default variant of a new product.
type ProductDraft struct {
	Name        string
	Description string
	Category    string
	UnitCode    string
	TaxRate     decimal.Decimal
	Attributes  map[string]domain.AttributeValue
	Variants    []VariantInput
}

// ExistingPolicy is what a draft does to a product whose SKUs the catalog
// already has.
type ExistingPolicy string

// The policies: update the product, or leave it as it is.
const (
	UpdateExisting ExistingPolicy = "update"
	SkipExisting   ExistingPolicy = "skip"
)

// DraftOutcome is what a draft does to the catalog.
type DraftOutcome string

// The outcomes.
const (
	DraftCreated DraftOutcome = "created"
	DraftUpdated DraftOutcome = "updated"
	DraftSkipped DraftOutcome = "skipped"
)

// DraftResult is what a draft did (or would do): its product and its
// variants in the draft's order, or Err, a domain error of the draft.
type DraftResult struct {
	ProductID uuid.UUID
	Outcome   DraftOutcome
	Variants  []DraftVariant
	Err       error
}

// DraftVariant is a variant of a draft in the catalog; Created is false
// for one that already existed.
type DraftVariant struct {
	ID      uuid.UUID
	Created bool
}

// CheckProductDrafts says what importing each draft would do, without
// writing anything. Every problem of a draft is its result's Err.
func (s *Service) CheckProductDrafts(ctx context.Context, a Actor, drafts []ProductDraft, p ExistingPolicy) (_ []DraftResult, err error) {
	ctx, span := s.startSpan(ctx, "CheckProductDrafts", a, attribute.Int("drafts", len(drafts)))
	defer func() { usecase.End(span, err) }()

	c, err := s.loadDraftCatalog(ctx, a, drafts)
	if err != nil {
		return nil, err
	}
	out := make([]DraftResult, 0, len(drafts))
	for _, d := range drafts {
		out = append(out, s.applyDraft(ctx, a, c, d, p, false))
	}
	return out, nil
}

// ImportProductDrafts writes the drafts in order and stops at the first one
// with an error, which is the last result. It does not undo the drafts
// written before: the caller runs it in a transaction and rolls it back.
// A category named by a draft that the tenant does not have is created.
func (s *Service) ImportProductDrafts(ctx context.Context, a Actor, drafts []ProductDraft, p ExistingPolicy) (_ []DraftResult, err error) {
	ctx, span := s.startSpan(ctx, "ImportProductDrafts", a, attribute.Int("drafts", len(drafts)))
	defer func() { usecase.End(span, err) }()

	c, err := s.loadDraftCatalog(ctx, a, drafts)
	if err != nil {
		return nil, err
	}
	out := make([]DraftResult, 0, len(drafts))
	for _, d := range drafts {
		r := s.applyDraft(ctx, a, c, d, p, true)
		out = append(out, r)
		if r.Err != nil {
			break
		}
	}
	return out, nil
}

// applyDraft creates, updates or skips the product d describes, writing it
// when save is true.
func (s *Service) applyDraft(ctx context.Context, a Actor, c *draftCatalog, d ProductDraft, p ExistingPolicy, save bool) DraftResult {
	existing, err := c.existing(d)
	switch {
	case err != nil:
		return DraftResult{Err: err}
	case existing == nil:
		return s.createDraft(ctx, a, c, d, save)
	case p == SkipExisting:
		return DraftResult{ProductID: existing.ID(), Outcome: DraftSkipped}
	default:
		return s.updateDraft(ctx, a, c, existing, d, save)
	}
}
