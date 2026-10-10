package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// loginCandidatesQuery calls the SECURITY DEFINER function of migration
// 00003. It is written by hand: sqlc cannot type the columns of a function
// returning TABLE.
const loginCandidatesQuery = `SELECT tenant_id, user_id FROM login_candidates($1)`

// LoginDirectory implements domain.LoginDirectory. It is the only read that
// crosses tenants, and it goes through login_candidates(), which returns ids
// only: the rows themselves are still read within their tenant.
type LoginDirectory struct{ db *db.DB }

// NewLoginDirectory returns a LoginDirectory on d.
func NewLoginDirectory(d *db.DB) *LoginDirectory { return &LoginDirectory{db: d} }

// FindByEmail returns the users registered with email, at most
// domain.MaxLoginCandidates, ordered by tenant id.
func (l *LoginDirectory) FindByEmail(ctx context.Context, email domain.Email) ([]domain.LoginCandidate, error) {
	var out []domain.LoginCandidate
	// Not a tenant transaction: the tenant is what it looks for.
	err := l.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, loginCandidatesQuery, email.String())
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LoginCandidate, error) {
			var c domain.LoginCandidate
			err := row.Scan(&c.TenantID, &c.UserID)
			return c, err
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("identity: find login candidates: %w", err)
	}
	return out, nil
}
