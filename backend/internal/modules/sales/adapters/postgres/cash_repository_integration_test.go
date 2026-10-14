//go:build integration

package postgres_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

var _ domain.CashSessionRepository = (*postgres.CashSessionRepository)(nil)

// openSession stores an open session of a new user at the tenant's location.
func openSession(t *testing.T, repo *postgres.CashSessionRepository, tn tenant, float string) *domain.CashSession {
	t.Helper()
	s, err := domain.OpenCashSession(domain.NewCashSessionParams{
		ID: newID(), TenantID: tn.id, LocationID: tn.location, UserID: newID(), OpeningFloat: dec(float), OpenedAt: now,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Open(t.Context(), s))
	return s
}

// saleIn is confirmed (total 123760.00, 80000 in cash and 50000 by Nequi:
// change 6240.00) sold by the session's cashier and charged in it.
func saleIn(t *testing.T, tn tenant, session *domain.CashSession) *domain.Sale {
	t.Helper()
	id := session.ID()
	return confirmedWith(t, tn, domain.NewSaleParams{
		ID: newID(), TenantID: tn.id, LocationID: session.LocationID(), UserID: session.UserID(), CashSessionID: &id, CreatedAt: now,
	})
}

func movementOf(s *domain.CashSession, kind domain.CashMovementKind, amount string) func(*domain.CashSession) (domain.CashMovement, error) {
	return func(locked *domain.CashSession) (domain.CashMovement, error) {
		return locked.RecordMovement(domain.CashMovementInput{
			ID: newID(), Kind: kind, Amount: dec(amount), Reason: "Motivo", UserID: s.UserID(), At: now,
		})
	}
}

func closeWith(s *domain.CashSession, pairs ...any) func(*domain.CashSession, domain.CashActivity) error {
	return func(locked *domain.CashSession, a domain.CashActivity) error {
		var counts []domain.DenominationCount
		for i := 0; i < len(pairs); i += 2 {
			counts = append(counts, domain.DenominationCount{Denomination: dec(pairs[i].(string)), Quantity: int64(pairs[i+1].(int))})
		}
		c, err := domain.NewCashCount(counts)
		if err != nil {
			return err
		}
		return locked.Close(domain.CloseCashInput{UserID: s.UserID(), Count: c, Activity: a, Notes: "Cierre", At: now.Add(time.Hour)})
	}
}

func TestCashSessionRepository(t *testing.T) {
	f := newFixture(t)
	repo := postgres.NewCashSessionRepository(f.db)

	t.Run("guarda y lee una caja abierta; es la caja actual de su usuario", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "150000.50")

		got, err := repo.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		want, snap := s.Snapshot(), got.Snapshot()
		want.OpeningFloat, snap.OpeningFloat = norm(want.OpeningFloat), norm(snap.OpeningFloat)
		require.Equal(t, want, snap)

		current, err := repo.Current(t.Context(), tn.id, s.UserID())
		require.NoError(t, err)
		require.Equal(t, s.ID(), current.ID())

		_, err = repo.Current(t.Context(), tn.id, newID())
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
	})

	t.Run("un usuario no puede tener dos cajas abiertas", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		second, err := domain.OpenCashSession(domain.NewCashSessionParams{
			ID: newID(), TenantID: tn.id, LocationID: tn.location, UserID: s.UserID(), OpeningFloat: dec("0"), OpenedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, repo.Open(t.Context(), second), domain.ErrCashSessionAlreadyOpen)
	})

	t.Run("rechaza abrir en una sede que no existe", func(t *testing.T) {
		tn := f.createTenant(t)
		s, err := domain.OpenCashSession(domain.NewCashSessionParams{
			ID: newID(), TenantID: tn.id, LocationID: newID(), UserID: newID(), OpeningFloat: dec("0"), OpenedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, repo.Open(t.Context(), s), domain.ErrLocationNotFound)
	})

	t.Run("registra y lista los movimientos en orden", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		in, err := repo.AddMovement(t.Context(), tn.id, s.ID(), movementOf(s, domain.CashIn, "20000"))
		require.NoError(t, err)
		out, err := repo.AddMovement(t.Context(), tn.id, s.ID(), movementOf(s, domain.CashOut, "5000.50"))
		require.NoError(t, err)

		got, err := repo.Movements(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, in.ID(), got[0].ID())
		require.Equal(t, out.ID(), got[1].ID())
		require.Equal(t, domain.CashOut, got[1].Kind())
		require.True(t, dec("5000.50").Equal(got[1].Amount()))
		require.Equal(t, "Motivo", got[1].Reason())
	})

	t.Run("el cierre suma solo las ventas confirmadas de la caja y sus movimientos", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "100000")
		other := openSession(t, repo, tn, "0")
		f.save(t, saleIn(t, tn, s))
		f.save(t, saleIn(t, tn, s))
		f.save(t, saleIn(t, tn, other))
		voided := saleIn(t, tn, s)
		f.save(t, voided)
		_, err := f.sales.Void(t.Context(), tn.id, voided.ID(), func(_ context.Context, sale *domain.Sale) error {
			return sale.Void(newID(), "Desistió", now)
		})
		require.NoError(t, err)
		_, err = repo.AddMovement(t.Context(), tn.id, s.ID(), movementOf(s, domain.CashIn, "10000"))
		require.NoError(t, err)
		_, err = repo.AddMovement(t.Context(), tn.id, s.ID(), movementOf(s, domain.CashOut, "3000"))
		require.NoError(t, err)

		// 100000 + 2 × 80000 + 10000 − 3000 − 2 × 6240 = 254520; counted 254500.
		closed, err := repo.Close(t.Context(), tn.id, s.ID(), closeWith(s, "100000", 2, "50000", 1, "2000", 2, "500", 1))
		require.NoError(t, err)

		c := closed.Closing()
		require.Equal(t, 2, c.Activity.Sales)
		require.True(t, dec("160000").Equal(c.Activity.CashReceived))
		require.True(t, dec("12480").Equal(c.Activity.ChangeGiven))
		require.True(t, dec("10000").Equal(c.Activity.CashIn))
		require.True(t, dec("3000").Equal(c.Activity.CashOut))
		require.True(t, dec("254520").Equal(c.Expected))
		require.True(t, dec("-20").Equal(c.Difference))
		require.Len(t, c.Activity.ByMethod, 2)
		require.Equal(t, domain.PaymentCash, c.Activity.ByMethod[0].Method)
		require.True(t, dec("147520").Equal(c.Activity.ByMethod[0].Amount), "efectivo neto del cambio")
		require.Equal(t, domain.PaymentNequi, c.Activity.ByMethod[1].Method)
		require.True(t, dec("100000").Equal(c.Activity.ByMethod[1].Amount))

		got, err := repo.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CashSessionClosed, got.Status())
		gc := got.Closing()
		require.True(t, dec("254520").Equal(gc.Expected))
		require.Len(t, gc.Count.Counts(), 4)
		require.True(t, dec("254500").Equal(gc.Count.Total()))
		require.Equal(t, "Cierre", gc.Notes)
		require.Len(t, gc.Activity.ByMethod, 2)

		_, err = repo.Current(t.Context(), tn.id, s.UserID())
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
	})

	t.Run("una venta anulada después del cierre no cambia la caja cerrada", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		sale := saleIn(t, tn, s)
		f.save(t, sale)
		_, err := repo.Close(t.Context(), tn.id, s.ID(), closeWith(s))
		require.NoError(t, err)
		_, err = f.sales.Void(t.Context(), tn.id, sale.ID(), func(_ context.Context, sale *domain.Sale) error {
			return sale.Void(newID(), "Desistió", now)
		})
		require.NoError(t, err)

		got, err := repo.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.True(t, dec("73760").Equal(got.Closing().Expected))
	})

	t.Run("una caja cerrada no se puede modificar", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		_, err := repo.Close(t.Context(), tn.id, s.ID(), closeWith(s))
		require.NoError(t, err)

		_, err = repo.Close(t.Context(), tn.id, s.ID(), closeWith(s, "1000", 1))
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
		_, err = repo.AddMovement(t.Context(), tn.id, s.ID(), movementOf(s, domain.CashIn, "1000"))
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)

		// Not even bypassing the domain: the database rejects it.
		err = f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "UPDATE cash_sessions SET notes = 'cambiado' WHERE id = $1", s.ID())
			return err
		})
		require.ErrorContains(t, err, "is closed")
		err = f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `INSERT INTO cash_movements (id, tenant_id, session_id, kind, amount, reason, user_id, created_at)
				VALUES ($1, $2, $3, 'out', 1000, 'x', $4, $5)`, newID(), tn.id, s.ID(), s.UserID(), now)
			return err
		})
		require.ErrorContains(t, err, "is not open")
	})

	t.Run("una venta no se guarda en una caja que se cerró", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		_, err := repo.Close(t.Context(), tn.id, s.ID(), closeWith(s))
		require.NoError(t, err)

		sale := saleIn(t, tn, s)
		err = f.sales.SaveConfirmed(t.Context(), sale, func(ctx context.Context) error {
			return repo.EnsureOpen(ctx, tn.id, s.ID())
		})
		require.ErrorIs(t, err, domain.ErrCashSessionClosed)
	})

	t.Run("el cierre espera a la venta que se está guardando y la cuenta", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		sale := saleIn(t, tn, s)

		locked := make(chan struct{})
		release := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := f.sales.SaveConfirmed(context.Background(), sale, func(ctx context.Context) error {
				if err := repo.EnsureOpen(ctx, tn.id, s.ID()); err != nil {
					return err
				}
				close(locked)
				<-release
				return nil
			})
			require.NoError(t, err)
		}()
		<-locked

		closed := make(chan *domain.CashSession, 1)
		go func() {
			c, err := repo.Close(context.Background(), tn.id, s.ID(), closeWith(s))
			require.NoError(t, err)
			closed <- c
		}()
		select {
		case <-closed:
			t.Fatal("el cierre no esperó a la venta")
		case <-time.After(200 * time.Millisecond):
		}
		close(release)
		wg.Wait()
		c := <-closed
		require.Equal(t, 1, c.Closing().Activity.Sales)
	})

	t.Run("aislamiento: la caja del tenant A no se ve ni se modifica desde el tenant B", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := openSession(t, repo, a, "0")

		_, err := repo.Get(t.Context(), b.id, s.ID())
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
		_, err = repo.Current(t.Context(), b.id, s.UserID())
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
		_, err = repo.AddMovement(t.Context(), b.id, s.ID(), movementOf(s, domain.CashIn, "1000"))
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
		_, err = repo.Close(t.Context(), b.id, s.ID(), closeWith(s))
		require.ErrorIs(t, err, domain.ErrCashSessionNotFound)
		require.ErrorIs(t, repo.EnsureOpen(t.Context(), b.id, s.ID()), domain.ErrCashSessionNotFound)
		movements, err := repo.Movements(t.Context(), b.id, s.ID())
		require.NoError(t, err)
		require.Empty(t, movements)

		got, err := repo.Get(t.Context(), a.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CashSessionOpen, got.Status())
	})

	t.Run("la venta guarda la caja donde se cobró", func(t *testing.T) {
		tn := f.createTenant(t)
		s := openSession(t, repo, tn, "0")
		sale := saleIn(t, tn, s)
		f.save(t, sale)

		got, err := f.sales.Get(t.Context(), tn.id, sale.ID())
		require.NoError(t, err)
		require.Equal(t, s.ID(), *got.CashSessionID())
	})
}

func TestPolicyRepositoryCashSession(t *testing.T) {
	f := newFixture(t)

	t.Run("sin configuración la caja es obligatoria", func(t *testing.T) {
		tn := f.createTenant(t)
		p, err := f.policies.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.False(t, p.CashSessionOptional)
	})

	t.Run("la empresa puede vender sin caja si lo configura", func(t *testing.T) {
		tn := f.createTenant(t)
		f.exec(t, tn.id, "INSERT INTO sales_settings (tenant_id, require_cash_session) VALUES ($1, false)", tn.id)
		p, err := f.policies.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.True(t, p.CashSessionOptional)
	})
}
