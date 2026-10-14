//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 10, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.SaleRepository   = (*postgres.SaleRepository)(nil)
	_ domain.PolicyRepository = (*postgres.PolicyRepository)(nil)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func noReserve(context.Context) error { return nil }

type fixture struct {
	db       *db.DB
	sales    *postgres.SaleRepository
	policies *postgres.PolicyRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, sales: postgres.NewSaleRepository(d), policies: postgres.NewPolicyRepository(d)}
}

func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// tenant is a tenant with a location and a product, stored directly: sales
// does not depend on the other modules, but its tables reference theirs.
type tenant struct {
	id, location, product uuid.UUID
}

func (f fixture) createTenant(t *testing.T) tenant {
	t.Helper()
	tn := tenant{id: newID(), product: newID()}
	f.exec(t, tn.id, "INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Ferretería', '890903938-8', 'active', $2)", tn.id, now)
	f.exec(t, tn.id, `INSERT INTO products (id, tenant_id, sku, name, description, base_unit, cost, price, tax_rate, active, created_at)
		VALUES ($1, $2, $3, 'Cemento gris 50kg', '', 'bulto', 25000, 30000, 0.19, true, $4)`,
		tn.product, tn.id, "SKU-"+strings.ToUpper(tn.product.String()), now)
	tn.location = f.createLocation(t, tn.id)
	return tn
}

func (f fixture) createLocation(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, tenantID, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, $3, true, $4)",
		id, tenantID, "Sede "+id.String(), now)
	return id
}

// confirmed builds a confirmed sale of the tenant's product at location, in
// two lines at 19%: 2.5 × 30000 with a discount of 1000 (base 74000, IVA
// 14060) and 1 × 30000 (IVA 5700); total 123760.00, paid with 50000 by Nequi
// and 80000 in cash.
func confirmed(t *testing.T, tn tenant, location uuid.UUID, confirmedAt time.Time) *domain.Sale {
	t.Helper()
	customer := newID()
	return confirmedWith(t, tn, domain.NewSaleParams{
		ID: newID(), TenantID: tn.id, LocationID: location, UserID: newID(), CustomerID: &customer, CreatedAt: confirmedAt,
	})
}

// confirmedWith is confirmed with the given sale fields.
func confirmedWith(t *testing.T, tn tenant, p domain.NewSaleParams) *domain.Sale {
	t.Helper()
	confirmedAt := p.CreatedAt
	s, err := domain.NewSale(p)
	require.NoError(t, err)
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: tn.product, ProductName: "Cemento gris 50kg", Quantity: dec("2.5"),
		UnitPrice: dec("30000"), TaxRate: dec("0.19"), Discount: dec("1000"),
	}, domain.NoDiscountLimit))
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: tn.product, ProductName: "Cemento gris 50kg", Quantity: dec("1"),
		UnitPrice: dec("30000"), TaxRate: dec("0.19"), Discount: dec("0"),
	}, domain.NoDiscountLimit))
	nequi, err := domain.NewPayment(domain.PaymentNequi, dec("50000"))
	require.NoError(t, err)
	cash, err := domain.NewPayment(domain.PaymentCash, dec("80000"))
	require.NoError(t, err)
	require.NoError(t, s.Confirm([]domain.Payment{nequi, cash}, confirmedAt))
	return s
}

// norm drops the trailing zeros that NUMERIC(18,4) adds (2.5000 → 2.5), so
// decimals of equal value compare equal.
func norm(d decimal.Decimal) decimal.Decimal { return decimal.RequireFromString(d.String()) }

// normalized returns snap with every decimal normalized.
func normalized(snap domain.SaleSnapshot) domain.SaleSnapshot {
	for i, l := range snap.Lines {
		snap.Lines[i] = domain.LineSnapshot{
			ProductID: l.ProductID, ProductName: l.ProductName, Quantity: norm(l.Quantity), UnitPrice: norm(l.UnitPrice),
			TaxRate: norm(l.TaxRate), Discount: norm(l.Discount), Gross: norm(l.Gross), Tax: norm(l.Tax),
		}
	}
	for i, p := range snap.Payments {
		snap.Payments[i].Amount = norm(p.Amount)
	}
	t := snap.Totals
	snap.Totals = domain.StoredTotals{
		Subtotal: norm(t.Subtotal), Discounts: norm(t.Discounts), Tax: norm(t.Tax),
		Total: norm(t.Total), Paid: norm(t.Paid), Change: norm(t.Change),
	}
	return snap
}

func (f fixture) save(t *testing.T, s *domain.Sale) {
	t.Helper()
	require.NoError(t, f.sales.SaveConfirmed(t.Context(), s, noReserve))
}

func TestSaleRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee una venta confirmada con sus líneas, pagos y consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)

		f.save(t, s)
		require.Equal(t, int64(1), s.Number())

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, normalized(s.Snapshot()), normalized(got.Snapshot()))
		require.Equal(t, int64(1), got.Number())
		require.Equal(t, s.CustomerID(), got.CustomerID())
	})

	t.Run("si reserve falla, la venta no queda guardada y lo que escribió reserve se deshace", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		boom := errors.New("sin stock")

		err := f.sales.SaveConfirmed(t.Context(), s, func(ctx context.Context) error {
			// reserve escribe en la misma transacción, como lo hace inventory.
			if err := f.db.WithTenantTx(ctx, tn.id, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO stock_levels (tenant_id, product_id, location_id, quantity, average_cost, updated_at)
					VALUES ($1, $2, $3, -2.5, 0, $4)`, tn.id, tn.product, tn.location, now)
				return err
			}); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)
		require.Zero(t, s.Number())

		_, err = f.sales.Get(t.Context(), tn.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
		var levels int
		require.NoError(t, f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels").Scan(&levels)
		}))
		require.Zero(t, levels)
	})

	t.Run("una venta que hace rollback no deja hueco en el consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		f.save(t, confirmed(t, tn, tn.location, now))
		require.Error(t, f.sales.SaveConfirmed(t.Context(), confirmed(t, tn, tn.location, now),
			func(context.Context) error { return errors.New("boom") }))

		third := confirmed(t, tn, tn.location, now)
		f.save(t, third)
		require.Equal(t, int64(2), third.Number())
	})

	t.Run("el consecutivo es independiente por sede", func(t *testing.T) {
		tn := f.createTenant(t)
		other := f.createLocation(t, tn.id)
		a1, b1, a2 := confirmed(t, tn, tn.location, now), confirmed(t, tn, other, now), confirmed(t, tn, tn.location, now)
		f.save(t, a1)
		f.save(t, b1)
		f.save(t, a2)
		require.Equal(t, []int64{1, 1, 2}, []int64{a1.Number(), b1.Number(), a2.Number()})
	})

	t.Run("dos ventas concurrentes en la misma sede no generan el mismo consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		const n = 12
		sales := make([]*domain.Sale, n)
		for i := range sales {
			sales[i] = confirmed(t, tn, tn.location, now)
		}

		var wg sync.WaitGroup
		errs := make([]error, n)
		for i, s := range sales {
			wg.Go(func() { errs[i] = f.sales.SaveConfirmed(context.Background(), s, noReserve) })
		}
		wg.Wait()

		numbers := make([]int64, 0, n)
		for i, s := range sales {
			require.NoError(t, errs[i])
			numbers = append(numbers, s.Number())
		}
		slices.Sort(numbers)
		want := make([]int64, n)
		for i := range want {
			want[i] = int64(i + 1)
		}
		require.Equal(t, want, numbers, "consecutivos sin repetir ni saltar")
	})

	t.Run("rechaza una sede de otro tenant", func(t *testing.T) {
		tn, other := f.createTenant(t), f.createTenant(t)
		s := confirmed(t, tn, other.location, now)
		err := f.sales.SaveConfirmed(t.Context(), s, noReserve)
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("anula una venta y corre change dentro de la transacción", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)
		by := newID()

		voided, err := f.sales.Void(t.Context(), tn.id, s.ID(), func(ctx context.Context, s *domain.Sale) error {
			return s.Void(by, "El cliente desistió", now.Add(time.Hour))
		})
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, voided.Status())

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, got.Status())
		require.Equal(t, &domain.VoidInfo{UserID: by, Reason: "El cliente desistió", At: now.Add(time.Hour)}, got.VoidInfo())
		require.Equal(t, int64(1), got.Number())
	})

	t.Run("si change falla, la venta sigue confirmada y lo que escribió se deshace", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)
		boom := errors.New("boom")

		_, err := f.sales.Void(t.Context(), tn.id, s.ID(), func(ctx context.Context, s *domain.Sale) error {
			if err := s.Void(newID(), "Error", now); err != nil {
				return err
			}
			if err := f.db.WithTenantTx(ctx, tn.id, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO stock_levels (tenant_id, product_id, location_id, quantity, average_cost, updated_at)
					VALUES ($1, $2, $3, 3.5, 0, $4)`, tn.id, tn.product, tn.location, now)
				return err
			}); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, got.Status())
		var levels int
		require.NoError(t, f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels").Scan(&levels)
		}))
		require.Zero(t, levels)
	})

	t.Run("dos anulaciones concurrentes devuelven el stock una sola vez", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)

		var reversals atomic.Int32
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := range errs {
			wg.Go(func() {
				_, errs[i] = f.sales.Void(context.Background(), tn.id, s.ID(), func(ctx context.Context, s *domain.Sale) error {
					if err := s.Void(newID(), "Duplicada", now); err != nil {
						return err
					}
					reversals.Add(1)
					// Mantiene el bloqueo un momento para que la otra anulación espere.
					time.Sleep(50 * time.Millisecond)
					return nil
				})
			})
		}
		wg.Wait()

		require.Equal(t, int32(1), reversals.Load())
		failed := 0
		for _, err := range errs {
			if err != nil {
				require.ErrorIs(t, err, domain.ErrSaleNotConfirmed)
				failed++
			}
		}
		require.Equal(t, 1, failed)
	})

	t.Run("anular una venta inexistente es sale_not_found", func(t *testing.T) {
		tn := f.createTenant(t)
		_, err := f.sales.Void(t.Context(), tn.id, newID(), func(context.Context, *domain.Sale) error { return nil })
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
	})

	t.Run("lista las ventas del rango de la más reciente a la más antigua, con sede y paginación", func(t *testing.T) {
		tn := f.createTenant(t)
		other := f.createLocation(t, tn.id)
		day := time.Date(2026, 10, 10, 5, 0, 0, 0, time.UTC)
		early := confirmed(t, tn, tn.location, day.Add(time.Hour))
		late := confirmed(t, tn, tn.location, day.Add(20*time.Hour))
		elsewhere := confirmed(t, tn, other, day.Add(2*time.Hour))
		before := confirmed(t, tn, tn.location, day.Add(-time.Second))
		next := confirmed(t, tn, tn.location, day.Add(24*time.Hour))
		for _, s := range []*domain.Sale{early, late, elsewhere, before, next} {
			f.save(t, s)
		}
		_, err := f.sales.Void(t.Context(), tn.id, early.ID(), func(_ context.Context, s *domain.Sale) error {
			return s.Void(newID(), "Error", day.Add(3*time.Hour))
		})
		require.NoError(t, err)

		page, err := f.sales.List(t.Context(), tn.id, domain.SaleFilter{From: day, To: day.Add(24 * time.Hour), Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 3, page.Total)
		ids := []uuid.UUID{page.Items[0].ID, page.Items[1].ID, page.Items[2].ID}
		require.Equal(t, []uuid.UUID{late.ID(), elsewhere.ID(), early.ID()}, ids)

		voided := page.Items[2]
		require.Equal(t, domain.SaleVoided, voided.Status)
		require.NotNil(t, voided.VoidedAt)
		require.True(t, dec("123760").Equal(voided.Total), "total %s", voided.Total)
		require.Equal(t, early.Number(), voided.Number)
		require.Equal(t, early.UserID(), voided.UserID)
		require.Equal(t, tn.location, voided.LocationID)
		require.Equal(t, day.Add(time.Hour), voided.ConfirmedAt)

		page, err = f.sales.List(t.Context(), tn.id, domain.SaleFilter{From: day, To: day.Add(24 * time.Hour), LocationID: &tn.location, Limit: 1, Offset: 1})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Len(t, page.Items, 1)
		require.Equal(t, early.ID(), page.Items[0].ID)
	})

	t.Run("rechaza un límite de página fuera de rango", func(t *testing.T) {
		tn := f.createTenant(t)
		for _, filter := range []domain.SaleFilter{{Limit: 0}, {Limit: 101}, {Limit: 10, Offset: -1}} {
			filter.From, filter.To = now, now.Add(time.Hour)
			_, err := f.sales.List(t.Context(), tn.id, filter)
			require.Error(t, err)
		}
	})

	t.Run("aislamiento: el tenant B no ve, no lista ni anula ventas del tenant A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := confirmed(t, a, a.location, now)
		f.save(t, s)

		_, err := f.sales.Get(t.Context(), b.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)

		page, err := f.sales.List(t.Context(), b.id, domain.SaleFilter{From: now.Add(-time.Hour), To: now.Add(time.Hour), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, page.Total)

		_, err = f.sales.Void(t.Context(), b.id, s.ID(), func(_ context.Context, s *domain.Sale) error {
			return s.Void(newID(), "Ajena", now)
		})
		require.ErrorIs(t, err, domain.ErrSaleNotFound)

		got, err := f.sales.Get(t.Context(), a.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, got.Status())
	})

	t.Run("aislamiento: el consecutivo del tenant B empieza en 1 aunque A ya vendió", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.save(t, confirmed(t, a, a.location, now))
		f.save(t, confirmed(t, a, a.location, now))
		s := confirmed(t, b, b.location, now)
		f.save(t, s)
		require.Equal(t, int64(1), s.Number())
	})
}

func TestPolicyRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("sin configuración el tope de descuento es 0", func(t *testing.T) {
		tn := f.createTenant(t)
		p, err := f.policies.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.False(t, p.MaxDiscount.Unlimited())
		require.True(t, p.MaxDiscount.Max().IsZero())
	})

	t.Run("devuelve el tope configurado solo a su tenant", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.exec(t, a.id, "INSERT INTO sales_settings (tenant_id, max_discount) VALUES ($1, 0.1)", a.id)

		p, err := f.policies.Get(t.Context(), a.id)
		require.NoError(t, err)
		require.True(t, dec("0.1").Equal(p.MaxDiscount.Max()))

		p, err = f.policies.Get(t.Context(), b.id)
		require.NoError(t, err)
		require.True(t, p.MaxDiscount.Max().IsZero())
	})
}
