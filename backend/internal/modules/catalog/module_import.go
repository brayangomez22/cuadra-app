// The part of the catalog module the imports module uses through
// internal/server. The import's API operation already checked the user's
// permission (imports:run).

package catalog

import (
	"context"

	"github.com/google/uuid"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// ImportColumns are what an import file's columns describe: the tenant's
// units, attribute definitions and variant axes.
type ImportColumns struct {
	Units      []*domain.UnitOfMeasure
	Attributes []*domain.AttributeDefinition
	Axes       []*domain.VariantAxis
}

// ImportColumns returns the tenant's units, attributes and axes.
func (m *Module) ImportColumns(ctx context.Context, tenantID, userID uuid.UUID) (ImportColumns, error) {
	a := app.Actor{TenantID: tenantID, UserID: userID}
	var c ImportColumns
	var err error
	if c.Units, err = m.service.ListUnits(ctx, a); err != nil {
		return ImportColumns{}, err
	}
	if c.Attributes, err = m.service.ListAttributes(ctx, a); err != nil {
		return ImportColumns{}, err
	}
	if c.Axes, err = m.service.ListVariantAxes(ctx, a); err != nil {
		return ImportColumns{}, err
	}
	return c, nil
}

// CheckProductDrafts says what importing each draft would do (see
// app.Service.CheckProductDrafts).
func (m *Module) CheckProductDrafts(ctx context.Context, tenantID, userID uuid.UUID, drafts []app.ProductDraft, p app.ExistingPolicy) ([]app.DraftResult, error) {
	return m.service.CheckProductDrafts(ctx, app.Actor{TenantID: tenantID, UserID: userID}, drafts, p)
}

// ImportProductDrafts writes the drafts (see app.Service.ImportProductDrafts);
// with a ctx from db.InTenantTx it joins the import's batch.
func (m *Module) ImportProductDrafts(ctx context.Context, tenantID, userID uuid.UUID, drafts []app.ProductDraft, p app.ExistingPolicy) ([]app.DraftResult, error) {
	return m.service.ImportProductDrafts(ctx, app.Actor{TenantID: tenantID, UserID: userID}, drafts, p)
}

// DescribeError returns the code and the Spanish message the API gives for
// a catalog error; ok is false for an error that is not the catalog's (an
// infrastructure failure).
func DescribeError(err error) (code, message string, ok bool) {
	return httpadapter.DescribeError(err)
}
