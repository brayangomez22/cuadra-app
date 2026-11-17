//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

var now = time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)

// queue records what the repository enqueued, or fails.
type queue struct {
	ids  []uuid.UUID
	fail error
}

func (q *queue) enqueue(_ context.Context, _ pgx.Tx, _ uuid.UUID, id uuid.UUID) error {
	if q.fail != nil {
		return q.fail
	}
	q.ids = append(q.ids, id)
	return nil
}

// seed stores a tenant and one of its users directly: feedback does not
// depend on identity, but its table references both.
func seed(t *testing.T, d *db.DB, name string) (tenant, user uuid.UUID) {
	t.Helper()
	tenant, user = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	require.NoError(t, d.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, $2, '900123456-8', 'active', 'generica', $3)",
			tenant, name, now); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(),
			"INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at) VALUES ($1, $2, 'ana@negocio.co', 'Ana', 'x', 'cashier', true, $3)",
			user, tenant, now)
		return err
	}))
	return tenant, user
}

func comment(t *testing.T, tenant, user uuid.UUID, message string) *domain.Feedback {
	t.Helper()
	f, err := domain.NewFeedback(tenant, user, message, "/vender", "1.4.0", now)
	require.NoError(t, err)
	return f
}

func TestFeedbackRepository(t *testing.T) {
	d := dbtest.New(t)

	t.Run("guarda el comentario, encola su aviso y lo marca notificado", func(t *testing.T) {
		q := &queue{}
		repo := postgres.NewFeedbackRepository(d, q.enqueue)
		tenant, user := seed(t, d, "Ferretería La Tuerca")
		f := comment(t, tenant, user, "El lector no lee los códigos cortos")

		require.NoError(t, repo.Create(t.Context(), f))
		require.Equal(t, []uuid.UUID{f.ID()}, q.ids)

		got, err := repo.Get(t.Context(), tenant, f.ID())
		require.NoError(t, err)
		require.Equal(t, "El lector no lee los códigos cortos", got.Message())
		require.False(t, got.Notified())

		got.MarkNotified(now.Add(time.Minute))
		require.NoError(t, repo.MarkNotified(t.Context(), got))
		got, err = repo.Get(t.Context(), tenant, f.ID())
		require.NoError(t, err)
		require.True(t, got.Notified())
	})

	t.Run("si no se puede encolar el aviso, el comentario no se guarda", func(t *testing.T) {
		repo := postgres.NewFeedbackRepository(d, (&queue{fail: errors.New("cola caída")}).enqueue)
		tenant, user := seed(t, d, "Calzado Andino")
		f := comment(t, tenant, user, "Quiero ver las tallas juntas")

		require.Error(t, repo.Create(t.Context(), f))
		_, err := repo.Get(t.Context(), tenant, f.ID())
		require.ErrorIs(t, err, domain.ErrFeedbackMissing)
	})

	t.Run("aislamiento: el tenant B no ve ni marca los comentarios del A", func(t *testing.T) {
		repo := postgres.NewFeedbackRepository(d, (&queue{}).enqueue)
		a, userA := seed(t, d, "Ferretería El Tornillo")
		b, _ := seed(t, d, "Droguería Central")
		f := comment(t, a, userA, "Bien")
		require.NoError(t, repo.Create(t.Context(), f))

		_, err := repo.Get(t.Context(), b, f.ID())
		require.ErrorIs(t, err, domain.ErrFeedbackMissing)

		var touched int64
		require.NoError(t, d.WithTenantTx(t.Context(), b, func(tx pgx.Tx) error {
			tag, err := tx.Exec(t.Context(), "UPDATE feedback SET notified_at = now() WHERE id = $1", f.ID())
			touched = tag.RowsAffected()
			return err
		}))
		require.Zero(t, touched)
		got, err := repo.Get(t.Context(), a, f.ID())
		require.NoError(t, err)
		require.False(t, got.Notified())
	})
}
