// Package app holds the catalog use cases: products and categories of a
// tenant.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Products       domain.ProductRepository
	Categories     domain.CategoryRepository
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the catalog use cases.
type Service struct {
	Deps
	tracer trace.Tracer

	productsCreated metric.Int64Counter
	priceChanges    metric.Int64Counter
	searches        metric.Int64Counter
}

// NewService builds a Service. It fails if a repository is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if d.Products == nil || d.Categories == nil {
		return nil, errors.New("catalog: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.TracerProvider == nil {
		d.TracerProvider = tracenoop.NewTracerProvider()
	}
	if d.MeterProvider == nil {
		d.MeterProvider = metricnoop.NewMeterProvider()
	}
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}

	s := &Service{Deps: d, tracer: d.TracerProvider.Tracer(instrumentationName)}
	meter := d.MeterProvider.Meter(instrumentationName)
	var err error
	if s.productsCreated, err = meter.Int64Counter("cuadra.catalog.products_created",
		metric.WithDescription("Products added to the catalog."),
		metric.WithUnit("{product}")); err != nil {
		return nil, err
	}
	if s.priceChanges, err = meter.Int64Counter("cuadra.catalog.price_changes",
		metric.WithDescription("Product price changes."),
		metric.WithUnit("{change}")); err != nil {
		return nil, err
	}
	if s.searches, err = meter.Int64Counter("cuadra.catalog.searches",
		metric.WithDescription("Product searches with a text, by result (hit, empty)."),
		metric.WithUnit("{search}")); err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	attrs = append(attrs,
		attribute.String("tenant.id", a.TenantID.String()),
		attribute.String("user.id", a.UserID.String()),
	)
	return s.tracer.Start(ctx, "catalog."+name, trace.WithAttributes(attrs...))
}

// endSpan records err on the span and ends it.
func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func productAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("product.id", id.String())
}

func categoryAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("category.id", id.String())
}
