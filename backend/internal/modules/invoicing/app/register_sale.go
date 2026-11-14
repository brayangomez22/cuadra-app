package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SaleDocumentRequest is a sale being confirmed that may need a document.
type SaleDocumentRequest struct {
	SaleID uuid.UUID
	// Kind is the kind the sale asks for ("pos" or "invoice"), or "".
	Kind        string
	Credit      bool
	HasCustomer bool
}

// RegisterSale gives a confirmed sale its electronic document, when the
// tenant issues them: it chooses the kind, draws the next number of the
// resolution in force and stores the pending document with its issue job.
// With the sale's ctx it all joins the sale's transaction, so the number is
// only taken if the sale is stored, and a sale without a resolution in force
// is not stored (domain.ErrNoActiveResolution). The provider is not called
// here: a sale is confirmed even when it is down.
func (s *Service) RegisterSale(ctx context.Context, a Actor, in SaleDocumentRequest) (err error) {
	ctx, span := s.startSpan(ctx, "RegisterSale", a, saleAttr(in.SaleID))
	defer func() { usecase.End(span, err) }()

	settings, err := s.Settings.Get(ctx, a.TenantID)
	if err != nil || !settings.Enabled {
		return err
	}
	kind, err := domain.ChooseKind(domain.KindChoice{
		Requested: domain.Kind(in.Kind), Credit: in.Credit, HasCustomer: in.HasCustomer, Default: settings.DefaultKind,
	})
	if err != nil {
		return err
	}
	now := s.Now()
	r, number, err := s.draw(ctx, a, kind, now)
	if err != nil {
		return err
	}
	doc, err := domain.NewDocument(domain.DocumentParams{
		ID: uuid.Must(uuid.NewV7()), TenantID: a.TenantID, SaleID: in.SaleID, CreatedBy: a.UserID,
		Kind: kind, Resolution: r, Number: number, IssuedAt: now,
	})
	if err != nil {
		return err
	}
	if err := s.Documents.Add(ctx, doc); err != nil {
		return err
	}
	span.SetAttributes(documentAttr(doc.ID()))
	s.countDocument(ctx, doc)
	return nil
}

// draw takes the next number of kind from the resolution in force, locked
// until the sale's transaction ends, and warns once when that number takes
// the resolution near its end.
func (s *Service) draw(ctx context.Context, a Actor, kind domain.Kind, now time.Time) (*domain.Resolution, int64, error) {
	rs, err := s.Resolutions.LockKind(ctx, a.TenantID, kind)
	if err != nil {
		return nil, 0, err
	}
	r, err := domain.PickResolution(rs, kind, now)
	if err != nil {
		return nil, 0, err
	}
	wasNear := r.NearEnd(now)
	number, err := r.Draw(now)
	if err != nil {
		return nil, 0, err
	}
	if err := s.Resolutions.SaveNext(ctx, r); err != nil {
		return nil, 0, err
	}
	if !wasNear && r.NearEnd(now) {
		s.nearEnd.Add(ctx, 1, metric.WithAttributes(kindAttr(kind)))
		s.Logger.WarnContext(ctx, "numbering resolution near its end", slog.String("resolution_id", r.ID().String()),
			slog.Int64("remaining", r.Remaining()), slog.String("valid_to", r.Params().ValidTo.String()))
	}
	return r, number, nil
}

// HasDocument is whether the sale has an electronic document; a sale with
// one is not voided, it gets a credit note (T37).
func (s *Service) HasDocument(ctx context.Context, a Actor, saleID uuid.UUID) (_ bool, err error) {
	ctx, span := s.startSpan(ctx, "HasDocument", a, saleAttr(saleID))
	defer func() { usecase.End(span, err) }()

	return s.Documents.ExistsForSale(ctx, a.TenantID, saleID)
}

// countDocument counts a document reaching its state.
func (s *Service) countDocument(ctx context.Context, d *domain.Document) {
	s.documents.Add(ctx, 1, metric.WithAttributes(kindAttr(d.Kind()), attribute.String("document.status", string(d.Status()))))
}

func kindAttr(k domain.Kind) attribute.KeyValue {
	return attribute.String("document.kind", string(k))
}
