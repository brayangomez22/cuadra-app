package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// UserRepository implements domain.UserRepository.
type UserRepository struct{ db *db.DB }

// NewUserRepository returns a UserRepository on d.
func NewUserRepository(d *db.DB) *UserRepository { return &UserRepository{db: d} }

// Create stores a new user; domain.ErrEmailTaken if the email is already
// registered in the tenant.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	err := r.db.WithTenantTx(ctx, u.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateUser(ctx, sqlcgen.CreateUserParams{
			ID:           u.ID(),
			TenantID:     u.TenantID(),
			Email:        u.Email().String(),
			Name:         u.Name(),
			PasswordHash: u.PasswordHash(),
			Role:         u.Role().String(),
			Active:       u.IsActive(),
			CreatedAt:    u.CreatedAt(),
		})
	})
	switch {
	case isUniqueViolation(err, usersEmailKey):
		return domain.ErrEmailTaken
	case err != nil:
		return fmt.Errorf("identity: create user: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrUserNotFound when the user does not exist in tenantID.
func (r *UserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	return r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.User, error) {
		return q.GetUser(ctx, sqlcgen.GetUserParams{TenantID: tenantID, ID: id})
	})
}

// GetByEmail returns domain.ErrUserNotFound when no user of tenantID has email.
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email domain.Email) (*domain.User, error) {
	return r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.User, error) {
		return q.GetUserByEmail(ctx, sqlcgen.GetUserByEmailParams{TenantID: tenantID, Email: email.String()})
	})
}

func (r *UserRepository) get(ctx context.Context, tenantID uuid.UUID, query func(*sqlcgen.Queries) (sqlcgen.User, error)) (*domain.User, error) {
	var row sqlcgen.User
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = query(sqlcgen.New(tx))
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrUserNotFound
	case err != nil:
		return nil, fmt.Errorf("identity: get user: %w", err)
	}
	return userFromRow(row)
}

// Update stores the name, password hash, role and active flag;
// domain.ErrUserNotFound if the user does not exist in its tenant.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, u.TenantID(), func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).UpdateUser(ctx, sqlcgen.UpdateUserParams{
			TenantID:     u.TenantID(),
			ID:           u.ID(),
			Name:         u.Name(),
			PasswordHash: u.PasswordHash(),
			Role:         u.Role().String(),
			Active:       u.IsActive(),
		})
		return err
	})
	switch {
	case err != nil:
		return fmt.Errorf("identity: update user: %w", err)
	case affected == 0:
		return domain.ErrUserNotFound
	}
	return nil
}

// userFromRow rebuilds a user. Errors name the user id only: the row holds
// personal data (email, name) that must not reach logs.
func userFromRow(row sqlcgen.User) (*domain.User, error) {
	email, err := domain.ParseEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("identity: user %s: stored email: %w", row.ID, err)
	}
	role, err := domain.ParseRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf("identity: user %s: stored role: %w", row.ID, err)
	}
	u, err := domain.RehydrateUser(domain.UserSnapshot{
		ID:           row.ID,
		TenantID:     row.TenantID,
		Email:        email,
		Name:         row.Name,
		PasswordHash: row.PasswordHash,
		Role:         role,
		Active:       row.Active,
		CreatedAt:    row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("identity: user %s: %w", row.ID, err)
	}
	return u, nil
}
