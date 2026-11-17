package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// ProviderAccountRepository implements domain.ProviderAccountRepository.
// The token reaches the database only encrypted by its cipher.
type ProviderAccountRepository struct {
	db     *db.DB
	cipher TokenCipher
}

// NewProviderAccountRepository returns a ProviderAccountRepository on d
// that encrypts the tokens with c.
func NewProviderAccountRepository(d *db.DB, c TokenCipher) *ProviderAccountRepository {
	return &ProviderAccountRepository{db: d, cipher: c}
}

// Get returns the tenant's account, or the default one if it has none. A
// token that does not open (another key) is an error that never carries it.
func (r *ProviderAccountRepository) Get(ctx context.Context, tenantID uuid.UUID) (domain.ProviderAccount, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetProviderAccountRow, error) {
		return sqlcgen.New(tx).GetProviderAccount(ctx, tenantID)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.DefaultProviderAccount(), nil
	case err != nil:
		return domain.ProviderAccount{}, fmt.Errorf("invoicing: get provider account: %w", err)
	}
	token, stale, err := r.token(tenantID, row.TokenCiphertext)
	if err != nil {
		return domain.ProviderAccount{}, err
	}
	a, err := domain.NewProviderAccount(row.Mode, row.TestSetID, token)
	if err != nil {
		return domain.ProviderAccount{}, fmt.Errorf("invoicing: tenant %s: stored provider account: %w", tenantID, err)
	}
	if stale {
		// Encrypted with the key before a rotation: store it with the
		// current one, so the previous key can be retired.
		if err := r.Save(ctx, tenantID, a); err != nil {
			return domain.ProviderAccount{}, err
		}
	}
	return a, nil
}

// token opens the stored token; none is the zero Secret. stale tells that
// it opened with the previous key.
func (r *ProviderAccountRepository) token(tenantID uuid.UUID, sealed []byte) (_ domain.Secret, stale bool, _ error) {
	if sealed == nil {
		return domain.Secret{}, false, nil
	}
	plain, stale, err := r.cipher.OpenRotating(tenantID, sealed)
	if err != nil {
		return domain.Secret{}, false, fmt.Errorf("invoicing: tenant %s: %w", tenantID, err)
	}
	s, err := domain.NewSecret(plain)
	if err != nil {
		return domain.Secret{}, false, fmt.Errorf("invoicing: tenant %s: stored token: %w", tenantID, err)
	}
	return s, stale, nil
}

// Save stores the tenant's account, with its token encrypted.
func (r *ProviderAccountRepository) Save(ctx context.Context, tenantID uuid.UUID, a domain.ProviderAccount) error {
	var sealed []byte
	if a.HasToken() {
		var err error
		if sealed, err = r.cipher.Seal(tenantID, a.Token.Reveal()); err != nil {
			return fmt.Errorf("invoicing: encrypt provider token: %w", err)
		}
	}
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).SaveProviderAccount(ctx, sqlcgen.SaveProviderAccountParams{
			TenantID: tenantID, Mode: string(a.Mode), TestSetID: a.TestSetID, TokenCiphertext: sealed,
		})
	})
	if err != nil {
		return fmt.Errorf("invoicing: save provider account: %w", err)
	}
	return nil
}
