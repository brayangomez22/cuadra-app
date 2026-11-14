package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// MunicipalityCatalog implements domain.MunicipalityCatalog. The catalog
// is reference data with no tenant: it is read outside a tenant's
// transaction, and the app can only read it.
type MunicipalityCatalog struct{ db *db.DB }

// NewMunicipalityCatalog returns a MunicipalityCatalog on d.
func NewMunicipalityCatalog(d *db.DB) *MunicipalityCatalog { return &MunicipalityCatalog{db: d} }

// List returns every municipality, by department and name.
func (c *MunicipalityCatalog) List(ctx context.Context) ([]domain.Municipality, error) {
	var rows []sqlcgen.Municipality
	err := c.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListMunicipalities(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("customers: list municipalities: %w", err)
	}
	out := make([]domain.Municipality, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Municipality{
			Code: r.Code, Name: r.Name, DepartmentCode: r.DepartmentCode, DepartmentName: r.DepartmentName,
		})
	}
	return out, nil
}
