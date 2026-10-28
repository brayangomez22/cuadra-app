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

// Level is a stock level with the product of its variant and its reorder
// rule, nil if the variant has none at the level's location.
type Level struct {
	*domain.StockLevel
	ProductID uuid.UUID
	Reorder   *domain.ReorderRule
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
	return withProductOf(ctx, s, a, movements, (*domain.StockMovement).VariantID,
		func(m *domain.StockMovement, product uuid.UUID) Movement {
			return Movement{StockMovement: m, ProductID: product}
		})
}

// levelsWithProducts adds to each level the product of its variant.
func (s *Service) levelsWithProducts(ctx context.Context, a Actor, levels []*domain.StockLevel) ([]Level, error) {
	return withProductOf(ctx, s, a, levels, (*domain.StockLevel).VariantID,
		func(l *domain.StockLevel, product uuid.UUID) Level { return Level{StockLevel: l, ProductID: product} })
}

// withProductOf builds a view of each item with the product of its variant,
// looked up in one call.
func withProductOf[T, V any](ctx context.Context, s *Service, a Actor, items []T,
	variant func(T) uuid.UUID, view func(T, uuid.UUID) V,
) ([]V, error) {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, variant(item))
	}
	products, err := s.ProductsOf(ctx, a, ids)
	if err != nil {
		return nil, err
	}
	out := make([]V, 0, len(items))
	for _, item := range items {
		out = append(out, view(item, products[variant(item)]))
	}
	return out, nil
}
