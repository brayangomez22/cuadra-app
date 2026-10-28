package app

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ImportTemplate returns an empty import file with the tenant's columns:
// one per attribute and, with the variants module on, one per axis.
func (s *Service) ImportTemplate(ctx context.Context, a Actor, format domain.Format) (_ []byte, err error) {
	ctx, span := s.startSpan(ctx, "ImportTemplate", a, attribute.String("import.format", string(format)))
	defer func() { usecase.End(span, err) }()

	layout, err := s.layout(ctx, a)
	if err != nil {
		return nil, err
	}
	return s.Sheets.Template(format, layout)
}
