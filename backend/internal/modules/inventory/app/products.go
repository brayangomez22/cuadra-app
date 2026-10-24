package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// Movement is a stock movement with the product of its variant, which the
// screens look the names up with.
type Movement struct {
	*domain.StockMovement
	ProductID uuid.UUID
}

// Level is a stock level with the product of its variant.
type Level struct {
	*domain.StockLevel
	ProductID uuid.UUID
}

// LevelPage is one page of levels and how many match in total.
type LevelPage struct {
	Items []Level
	Total int
}

// MovementPage is one page of movements and how many match in total.
type MovementPage struct {
	Items []Movement
	Total int
}

// withProducts adds to each movement the product of its variant.
func (s *Service) withProducts(ctx context.Context, a Actor, movements ...*domain.StockMovement) ([]Movement, error) {
	ids := make([]uuid.UUID, 0, len(movements))
	for _, m := range movements {
		ids = append(ids, m.VariantID())
	}
	products, err := s.ProductsOf(ctx, a, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Movement, 0, len(movements))
	for _, m := range movements {
		out = append(out, Movement{StockMovement: m, ProductID: products[m.VariantID()]})
	}
	return out, nil
}

// levelsWithProducts adds to each level the product of its variant.
func (s *Service) levelsWithProducts(ctx context.Context, a Actor, levels []*domain.StockLevel) ([]Level, error) {
	ids := make([]uuid.UUID, 0, len(levels))
	for _, l := range levels {
		ids = append(ids, l.VariantID())
	}
	products, err := s.ProductsOf(ctx, a, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Level, 0, len(levels))
	for _, l := range levels {
		out = append(out, Level{StockLevel: l, ProductID: products[l.VariantID()]})
	}
	return out, nil
}
