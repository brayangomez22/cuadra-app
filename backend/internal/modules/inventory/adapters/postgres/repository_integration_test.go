//go:build integration

package postgres_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 10, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.LocationRepository = (*postgres.LocationRepository)(nil)
	_ domain.StockRepository    = (*postgres.StockRepository)(nil)
	_ domain.PolicyRepository   = (*postgres.PolicyRepository)(nil)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

type fixture struct {
	db        *db.DB
	locations *postgres.LocationRepository
	stock     *postgres.StockRepository
	policies  *postgres.PolicyRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{
		db:        d,
		locations: postgres.NewLocationRepository(d),
		stock:     postgres.NewStockRepository(d),
		policies:  postgres.NewPolicyRepository(d),
	}
}

// exec runs sql in a transaction scoped to tenantID.
func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// createTenant stores a tenant row directly: inventory does not depend on
// the identity module, but its tables reference tenants.
func (f fixture) createTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, id, "INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Ferretería', '890903938-8', 'active', $2)", id, now)
	return id
}

// createProduct stores a product row directly, for the same reason.
func (f fixture) createProduct(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, tenantID, `INSERT INTO products (id, tenant_id, sku, name, description, base_unit, cost, price, tax_rate, active, created_at)
		VALUES ($1, $2, $3, 'Cemento gris 50kg', '', 'bulto', 30000, 35000, 0.19, true, $4)`,
		id, tenantID, "SKU-"+strings.ToUpper(id.String()), now)
	return id
}

func (f fixture) createLocation(t *testing.T, tenantID uuid.UUID, name string) *domain.Location {
	t.Helper()
	l, err := domain.NewLocation(tenantID, name, now)
	require.NoError(t, err)
	require.NoError(t, f.locations.Create(t.Context(), l))
	return l
}

func info(userID uuid.UUID) domain.MovementInfo {
	return domain.MovementInfo{UserID: userID, OccurredAt: now}
}

// receive registers an entry through Apply, as the use case does.
func (f fixture) receive(t *testing.T, tenantID, productID, locationID uuid.UUID, qty, cost string) {
	t.Helper()
	err := f.stock.Apply(t.Context(), tenantID, productID, []uuid.UUID{locationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			m, err := levels[0].Receive(dec(qty), dec(cost), info(newID()))
			return []*domain.StockMovement{m}, err
		})
	require.NoError(t, err)
}

// level returns the stored level of a product at a location; zero if none.
func (f fixture) level(t *testing.T, tenantID, productID, locationID uuid.UUID) decimal.Decimal {
	t.Helper()
	page, err := f.stock.ListLevels(t.Context(), tenantID, domain.StockFilter{LocationID: locationID, ProductID: &productID, Limit: 10})
	require.NoError(t, err)
	if page.Total == 0 {
		return decimal.Zero
	}
	return page.Items[0].Quantity()
}

func TestLocationRepository(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)

	t.Run("crea, lee y lista sedes por nombre", func(t *testing.T) {
		tenant := f.createTenant(t)
		north := f.createLocation(t, tenant, "Sede norte")
		f.createLocation(t, tenant, "Bodega")

		got, err := f.locations.GetByID(t.Context(), tenant, north.ID())
		require.NoError(t, err)
		require.Equal(t, "Sede norte", got.Name())
		require.True(t, got.IsActive())
		require.Equal(t, now, got.CreatedAt())

		list, err := f.locations.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Bodega", list[0].Name())
	})

	t.Run("rechaza nombre repetido sin importar mayúsculas", func(t *testing.T) {
		f.createLocation(t, tenant, "Bodega principal")
		l, err := domain.NewLocation(tenant, "BODEGA principal", now)
		require.NoError(t, err)
		require.ErrorIs(t, f.locations.Create(t.Context(), l), domain.ErrLocationNameTaken)
	})

	t.Run("una sede inexistente da ErrLocationNotFound", func(t *testing.T) {
		_, err := f.locations.GetByID(t.Context(), tenant, newID())
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})
}

func TestPolicyRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("sin fila de configuración, la política prohíbe stock negativo", func(t *testing.T) {
		tenant := f.createTenant(t)
		policy, err := f.policies.Get(t.Context(), tenant)
		require.NoError(t, err)
		require.False(t, policy.AllowNegative)
	})

	t.Run("lee la política guardada del tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.exec(t, tenant, "INSERT INTO inventory_settings (tenant_id, allow_negative_stock) VALUES ($1, true)", tenant)
		policy, err := f.policies.Get(t.Context(), tenant)
		require.NoError(t, err)
		require.True(t, policy.AllowNegative)
	})
}

func TestStockRepositoryApply(t *testing.T) {
	f := newFixture(t)

	t.Run("dos salidas simultáneas por el total del stock: solo una pasa", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, product, loc.ID(), "10", "1000")

		const workers = 8
		start := make(chan struct{})
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				<-start
				errs[i] = f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{loc.ID()},
					func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
						m, err := levels[0].Issue(dec("10"), domain.StockPolicy{}, info(newID()))
						return []*domain.StockMovement{m}, err
					})
			})
		}
		close(start)
		wg.Wait()

		var ok, insufficient int
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, domain.ErrInsufficientStock):
				insufficient++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		require.Equal(t, 1, ok)
		require.Equal(t, workers-1, insufficient)
		require.True(t, f.level(t, tenant, product, loc.ID()).IsZero())
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{ProductID: product, Limit: 100})
		require.NoError(t, err)
		require.Equal(t, 2, kardex.Total)
	})

	t.Run("dos traslados opuestos simultáneos no se bloquean entre sí", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		a := f.createLocation(t, tenant, "Sede A")
		b := f.createLocation(t, tenant, "Sede B")
		f.receive(t, tenant, product, a.ID(), "100", "1000")
		f.receive(t, tenant, product, b.ID(), "100", "1000")

		transfer := func(from, to uuid.UUID) error {
			return f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{from, to},
				func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
					out, in, err := domain.Transfer(levels[0], levels[1], dec("1"), domain.StockPolicy{}, newID(), now)
					return []*domain.StockMovement{out, in}, err
				})
		}
		const rounds = 20
		start := make(chan struct{})
		errs := make(chan error, 2*rounds)
		var wg sync.WaitGroup
		for range rounds {
			wg.Go(func() { <-start; errs <- transfer(a.ID(), b.ID()) })
			wg.Go(func() { <-start; errs <- transfer(b.ID(), a.ID()) })
		}
		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
		}
		require.True(t, dec("100").Equal(f.level(t, tenant, product, a.ID())))
		require.True(t, dec("100").Equal(f.level(t, tenant, product, b.ID())))
	})

	t.Run("una operación fallida no deja niveles ni movimientos", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{loc.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Issue(dec("1"), domain.StockPolicy{}, info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		levels, err := f.stock.ListLevels(t.Context(), tenant, domain.StockFilter{LocationID: loc.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, levels.Total)
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{ProductID: product, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("un producto inexistente da ErrProductNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, newID(), []uuid.UUID{loc.ID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("una sede inexistente da ErrLocationNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)

		err := f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{newID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("entrega los niveles en el orden de las sedes pedidas", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		first := f.createLocation(t, tenant, "Primera")
		second := f.createLocation(t, tenant, "Segunda")

		err := f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{second.ID(), first.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				require.Equal(t, second.ID(), levels[0].LocationID())
				require.Equal(t, first.ID(), levels[1].LocationID())
				return nil, nil
			})
		require.NoError(t, err)
	})
}

func TestKardex(t *testing.T) {
	f := newFixture(t)

	t.Run("el kardex reconstruye el stock actual", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()

		f.receive(t, tenant, product, bodega.ID(), "10", "1000")
		f.receive(t, tenant, product, bodega.ID(), "2.5", "1200")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1.25"), "Bulto roto", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		require.NoError(t, f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("3"), domain.StockPolicy{}, user, now)
				return []*domain.StockMovement{out, in}, err
			}))

		for _, loc := range []*domain.Location{bodega, centro} {
			locID := loc.ID()
			kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{ProductID: product, LocationID: &locID, Limit: 100})
			require.NoError(t, err)
			sum := decimal.Zero
			for _, m := range kardex.Items {
				require.Equal(t, locID, m.LocationID())
				sum = sum.Add(m.Quantity())
			}
			current := f.level(t, tenant, product, locID)
			require.True(t, sum.Equal(current), "%s: kardex %s, level %s", loc.Name(), sum, current)
			require.True(t, kardex.Items[0].BalanceAfter().Equal(current), "the newest movement carries the balance")
		}
		require.True(t, dec("8.25").Equal(f.level(t, tenant, product, bodega.ID())))
		require.True(t, dec("3").Equal(f.level(t, tenant, product, centro.ID())))
	})

	t.Run("devuelve los movimientos completos, el más reciente primero y paginados", func(t *testing.T) {
		tenant := f.createTenant(t)
		product := f.createProduct(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()
		f.receive(t, tenant, product, bodega.ID(), "10", "1000")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1"), "Conteo físico", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		var transferOut *domain.StockMovement
		require.NoError(t, f.stock.Apply(t.Context(), tenant, product, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("2"), domain.StockPolicy{}, user, now)
				transferOut = out
				return []*domain.StockMovement{out, in}, err
			}))

		page, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{ProductID: product, Limit: 2, Offset: 1})
		require.NoError(t, err)
		require.Equal(t, 4, page.Total)
		require.Len(t, page.Items, 2)
		got := page.Items[0]
		require.Equal(t, transferOut.ID(), got.ID())
		require.Equal(t, domain.MovementTransferOut, got.Type())
		require.Equal(t, transferOut.Reference(), got.Reference())
		require.True(t, dec("-2").Equal(got.Quantity()))
		require.True(t, dec("1000").Equal(got.UnitCost()))
		require.True(t, dec("7").Equal(got.BalanceAfter()))
		require.True(t, dec("1000").Equal(got.AverageCostAfter()))
		require.Equal(t, user, got.UserID())
		require.Equal(t, now, got.OccurredAt())
		adjustment := page.Items[1]
		require.Equal(t, domain.MovementAdjustment, adjustment.Type())
		require.Equal(t, "Conteo físico", adjustment.Reason())
		require.Nil(t, adjustment.Reference())
	})
}

func TestTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA := f.createTenant(t)
	tenantB := f.createTenant(t)
	productA := f.createProduct(t, tenantA)
	productB := f.createProduct(t, tenantB)
	locA := f.createLocation(t, tenantA, "Bodega")
	locB := f.createLocation(t, tenantB, "Bodega")
	f.receive(t, tenantA, productA, locA.ID(), "10", "1000")

	t.Run("aislamiento: el tenant B no ve sedes, niveles ni kardex del tenant A", func(t *testing.T) {
		_, err := f.locations.GetByID(t.Context(), tenantB, locA.ID())
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		list, err := f.locations.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, locB.ID(), list[0].ID())

		levels, err := f.stock.ListLevels(t.Context(), tenantB, domain.StockFilter{LocationID: locA.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, levels.Total)
		kardex, err := f.stock.Kardex(t.Context(), tenantB, domain.KardexFilter{ProductID: productA, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("aislamiento: el tenant B no puede registrar movimientos sobre un producto del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, productA, []uuid.UUID{locB.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("aislamiento: el tenant B no puede mover stock en una sede del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, productB, []uuid.UUID{locA.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		require.True(t, dec("10").Equal(f.level(t, tenantA, productA, locA.ID())))
	})

	t.Run("aislamiento: un UPDATE desde el tenant B no cambia los niveles del tenant A", func(t *testing.T) {
		f.exec(t, tenantB, "UPDATE stock_levels SET quantity = 0")
		require.True(t, dec("10").Equal(f.level(t, tenantA, productA, locA.ID())))
	})
}

func TestMovementsAreImmutable(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	product := f.createProduct(t, tenant)
	loc := f.createLocation(t, tenant, "Bodega")
	f.receive(t, tenant, product, loc.ID(), "10", "1000")

	for _, sql := range []string{
		"UPDATE stock_movements SET quantity = 99",
		"DELETE FROM stock_movements",
	} {
		t.Run("app_user no puede modificar ni borrar movimientos: "+sql, func(t *testing.T) {
			err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
				_, err := tx.Exec(t.Context(), sql)
				return err
			})
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, "42501", pgErr.Code, "insufficient_privilege")
		})
	}
}
