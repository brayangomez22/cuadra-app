package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CountEntryInput is what someone counted of a variant: Quantity is in
// UnitID, an alternate unit of the variant, or in its base unit when UnitID
// is nil.
type CountEntryInput struct {
	CountID   uuid.UUID
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	UnitID    *uuid.UUID
}

// EntryView is an entry with the product of its variant, to show its name.
type EntryView struct {
	*domain.CountEntry
	ProductID uuid.UUID
}

// EntryPage is one page of entries and how many match in total.
type EntryPage struct {
	Items []EntryView
	Total int
}

// RecordCountEntry stores what the actor counted, in the base unit. In a
// full count, a variant without a line (it had no stock at the start) gets
// one starting at zero; a partial count only takes its own variants.
func (s *Service) RecordCountEntry(ctx context.Context, a Actor, in CountEntryInput) (_ EntryView, err error) {
	ctx, span := s.startSpan(ctx, "RecordCountEntry", a, countAttr(in.CountID), variantAttr(in.VariantID),
		attribute.Bool("unit.alternate", in.UnitID != nil))
	defer func() { usecase.End(span, err) }()

	c, err := s.Counts.Get(ctx, a.TenantID, in.CountID)
	if err != nil {
		return EntryView{}, err
	}
	if in.UnitID != nil {
		factor, err := s.Variants.UnitFactor(ctx, a.TenantID, in.VariantID, *in.UnitID)
		if err != nil {
			return EntryView{}, err
		}
		if in.Quantity, _, err = domain.ToBaseUnit(in.Quantity, decimal.Zero, factor); err != nil {
			return EntryView{}, err
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return EntryView{}, err
	}
	e, err := c.NewEntry(id, in.VariantID, in.Quantity, a.UserID, s.Now())
	if err != nil {
		return EntryView{}, err
	}
	if err := s.CountEntries.Add(ctx, e, !c.IsPartial()); err != nil {
		return EntryView{}, err
	}
	s.countEntries.Add(ctx, 1)
	views, err := s.entryViews(ctx, a, []*domain.CountEntry{e})
	if err != nil {
		return EntryView{}, err
	}
	return views[0], nil
}

// DeleteCountEntry removes an entry of an open count: the actor's own, or
// anyone's if the actor manages counts.
func (s *Service) DeleteCountEntry(ctx context.Context, a Actor, countID, entryID uuid.UUID, manages bool) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteCountEntry", a, countAttr(countID), attribute.String("entry.id", entryID.String()))
	defer func() { usecase.End(span, err) }()

	c, err := s.Counts.Get(ctx, a.TenantID, countID)
	if err != nil {
		return err
	}
	e, err := s.CountEntries.Get(ctx, a.TenantID, countID, entryID)
	if err != nil {
		return err
	}
	if err := c.CheckEntryRemoval(e, a.UserID, manages); err != nil {
		return err
	}
	return s.CountEntries.Delete(ctx, e)
}

// ListCountEntries returns one page of the entries of a count in their
// current round: only the actor's own unless the actor manages counts.
func (s *Service) ListCountEntries(ctx context.Context, a Actor, f domain.EntryFilter, manages bool) (_ EntryPage, err error) {
	ctx, span := s.startSpan(ctx, "ListCountEntries", a, countAttr(f.CountID))
	defer func() { usecase.End(span, err) }()

	if _, err := s.Counts.Get(ctx, a.TenantID, f.CountID); err != nil {
		return EntryPage{}, err
	}
	if !manages {
		f.CountedBy = &a.UserID
	}
	page, err := s.CountEntries.List(ctx, a.TenantID, f)
	if err != nil {
		return EntryPage{}, err
	}
	views, err := s.entryViews(ctx, a, page.Items)
	if err != nil {
		return EntryPage{}, err
	}
	return EntryPage{Items: views, Total: page.Total}, nil
}

// entryViews adds the product of each entry's variant.
func (s *Service) entryViews(ctx context.Context, a Actor, entries []*domain.CountEntry) ([]EntryView, error) {
	return withProductOf(ctx, s, a, entries, (*domain.CountEntry).VariantID,
		func(e *domain.CountEntry, product uuid.UUID) EntryView {
			return EntryView{CountEntry: e, ProductID: product}
		})
}
