package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// productsPerSearch is how many products BySKU loads per Search: a page's
// maximum.
const productsPerSearch = 100

// BySKU returns the tenant's products with a variant whose SKU is among
// skus (compared as stored: trimmed and in uppercase), with all their
// variants. With a ctx from db.InTenantTx it reads in that transaction, so
// it sees what an import's batch already wrote.
func (r *ProductRepository) BySKU(ctx context.Context, tenantID uuid.UUID, skus []string) ([]*domain.Product, error) {
	stored := make([]string, 0, len(skus))
	for _, s := range skus {
		stored = append(stored, strings.ToUpper(strings.TrimSpace(s)))
	}
	ids, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]uuid.UUID, error) {
		return sqlcgen.New(tx).SKUProducts(ctx, stored)
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: products by SKU: %w", err)
	}
	out := make([]*domain.Product, 0, len(ids))
	for start := 0; start < len(ids); start += productsPerSearch {
		chunk := ids[start:min(start+productsPerSearch, len(ids))]
		page, err := r.Search(ctx, tenantID, domain.ProductFilter{IDs: chunk, Limit: len(chunk)})
		if err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
	}
	return out, nil
}
