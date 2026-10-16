//go:build integration

package postgres_test

import (
	"errors"
	"strconv"
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

// createProduct stores a product row and its variants directly, for the same
// reason: the default variant and extra more. Like a product created after
// variants existed, no variant has the product's id. It returns the
// variant ids, the default one first.
func (f fixture) createProduct(t *testing.T, tenantID uuid.UUID, extra int) []uuid.UUID {
	t.Helper()
	productID := newID()
	f.exec(t, tenantID, `INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
		VALUES ($1, $2, 'und', 'Unidad', $3) ON CONFLICT (tenant_id, code) DO NOTHING`, newID(), tenantID, now)
	f.exec(t, tenantID, `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
		VALUES ($1, $2, 'Producto', '', (SELECT id FROM units_of_measure WHERE code = 'und'), 0.19, true, $3)`,
		productID, tenantID, now)
	ids := make([]uuid.UUID, 0, extra+1)
	for i := range extra + 1 {
		id := newID()
		f.exec(t, tenantID, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
			VALUES ($1, $2, $3, $4, 30000, 35000, jsonb_build_object('opcion', $5::text), $6, true, $7)`,
			id, tenantID, productID, "SKU-"+strings.ToUpper(id.String()), strconv.Itoa(i), i == 0, now)
		ids = append(ids, id)
	}
	return ids
}

// createVariant stores a simple product and returns its default variant.
func (f fixture) createVariant(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return f.createProduct(t, tenantID, 0)[0]
}

// levelRows counts the stored levels of a variant at a location, lot or not.
func (f fixture) levelRows(t *testing.T, tenantID, variantID, locationID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels WHERE variant_id = $1 AND location_id = $2",
			variantID, locationID).Scan(&n)
	}))
	return n
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
func (f fixture) receive(t *testing.T, tenantID, variantID, locationID uuid.UUID, qty, cost string) {
	t.Helper()
	err := f.stock.Apply(t.Context(), tenantID, variantID, []uuid.UUID{locationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			m, err := levels[0].Receive(dec(qty), dec(cost), info(newID()))
			return []*domain.StockMovement{m}, err
		})
	require.NoError(t, err)
}

// level returns the stored level of a variant at a location; zero if none.
func (f fixture) level(t *testing.T, tenantID, variantID, locationID uuid.UUID) decimal.Decimal {
	t.Helper()
	page, err := f.stock.ListLevels(t.Context(), tenantID, domain.StockFilter{LocationID: locationID, VariantID: &variantID, Limit: 10})
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
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")

		const workers = 8
		start := make(chan struct{})
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				<-start
				errs[i] = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
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
		require.True(t, f.level(t, tenant, variant, loc.ID()).IsZero())
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 100})
		require.NoError(t, err)
		require.Equal(t, 2, kardex.Total)
	})

	t.Run("dos traslados opuestos simultáneos no se bloquean entre sí", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		a := f.createLocation(t, tenant, "Sede A")
		b := f.createLocation(t, tenant, "Sede B")
		f.receive(t, tenant, variant, a.ID(), "100", "1000")
		f.receive(t, tenant, variant, b.ID(), "100", "1000")

		transfer := func(from, to uuid.UUID) error {
			return f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{from, to},
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
		require.True(t, dec("100").Equal(f.level(t, tenant, variant, a.ID())))
		require.True(t, dec("100").Equal(f.level(t, tenant, variant, b.ID())))
	})

	t.Run("una operación fallida no deja niveles ni movimientos", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Issue(dec("1"), domain.StockPolicy{}, info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		levels, err := f.stock.ListLevels(t.Context(), tenant, domain.StockFilter{LocationID: loc.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, levels.Total)
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("una variante inexistente da ErrVariantNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, newID(), []uuid.UUID{loc.ID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})

	t.Run("una sede inexistente da ErrLocationNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{newID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("entrega los niveles en el orden de las sedes pedidas", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		first := f.createLocation(t, tenant, "Primera")
		second := f.createLocation(t, tenant, "Segunda")

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{second.ID(), first.ID()},
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
		variant := f.createVariant(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()

		f.receive(t, tenant, variant, bodega.ID(), "10", "1000")
		f.receive(t, tenant, variant, bodega.ID(), "2.5", "1200")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1.25"), "Bulto roto", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("3"), domain.StockPolicy{}, user, now)
				return []*domain.StockMovement{out, in}, err
			}))

		for _, loc := range []*domain.Location{bodega, centro} {
			locID := loc.ID()
			kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, LocationID: &locID, Limit: 100})
			require.NoError(t, err)
			sum := decimal.Zero
			for _, m := range kardex.Items {
				require.Equal(t, locID, m.LocationID())
				sum = sum.Add(m.Quantity())
			}
			current := f.level(t, tenant, variant, locID)
			require.True(t, sum.Equal(current), "%s: kardex %s, level %s", loc.Name(), sum, current)
			require.True(t, kardex.Items[0].BalanceAfter().Equal(current), "the newest movement carries the balance")
		}
		require.True(t, dec("8.25").Equal(f.level(t, tenant, variant, bodega.ID())))
		require.True(t, dec("3").Equal(f.level(t, tenant, variant, centro.ID())))
	})

	t.Run("devuelve los movimientos completos, el más reciente primero y paginados", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()
		f.receive(t, tenant, variant, bodega.ID(), "10", "1000")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1"), "Conteo físico", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		var transferOut *domain.StockMovement
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("2"), domain.StockPolicy{}, user, now)
				transferOut = out
				return []*domain.StockMovement{out, in}, err
			}))

		page, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 2, Offset: 1})
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
	variantA := f.createVariant(t, tenantA)
	variantB := f.createVariant(t, tenantB)
	locA := f.createLocation(t, tenantA, "Bodega")
	locB := f.createLocation(t, tenantB, "Bodega")
	f.receive(t, tenantA, variantA, locA.ID(), "10", "1000")

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
		kardex, err := f.stock.Kardex(t.Context(), tenantB, domain.KardexFilter{VariantID: variantA, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("aislamiento: el tenant B no puede registrar movimientos sobre una variante del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, variantA, []uuid.UUID{locB.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})

	t.Run("aislamiento: el tenant B no puede mover stock en una sede del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, variantB, []uuid.UUID{locA.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		require.True(t, dec("10").Equal(f.level(t, tenantA, variantA, locA.ID())))
	})

	t.Run("aislamiento: un UPDATE desde el tenant B no cambia los niveles del tenant A", func(t *testing.T) {
		f.exec(t, tenantB, "UPDATE stock_levels SET quantity = 0")
		require.True(t, dec("10").Equal(f.level(t, tenantA, variantA, locA.ID())))
	})
}

func TestMovementsAreImmutable(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	variant := f.createVariant(t, tenant)
	loc := f.createLocation(t, tenant, "Bodega")
	f.receive(t, tenant, variant, loc.ID(), "10", "1000")

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

func TestStockRepositorySaleMovements(t *testing.T) {
	f := newFixture(t)

	sell := func(t *testing.T, tenantID, variantID, locationID uuid.UUID, ref domain.DocumentRef, qty string) {
		t.Helper()
		err := f.stock.Apply(t.Context(), tenantID, variantID, []uuid.UUID{locationID},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				i := info(newID())
				i.Reference = &ref
				m, err := levels[0].Issue(dec(qty), domain.StockPolicy{}, i)
				return []*domain.StockMovement{m}, err
			})
		require.NoError(t, err)
	}

	t.Run("lista los movimientos de una venta y guarda su sale_void", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")
		sale := domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
		sell(t, tenant, variant, loc.ID(), sale, "3")
		sell(t, tenant, variant, loc.ID(), domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}, "1")

		got, err := f.stock.MovementsByReference(t.Context(), tenant, sale)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, domain.MovementSaleOut, got[0].Type())
		require.True(t, dec("-3").Equal(got[0].Quantity()))

		err = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				i := info(newID())
				i.Reference = &sale
				m, err := levels[0].ReturnSale(dec("3"), got[0].UnitCost(), i)
				return []*domain.StockMovement{m}, err
			})
		require.NoError(t, err)
		require.True(t, dec("9").Equal(f.level(t, tenant, variant, loc.ID())))

		got, err = f.stock.MovementsByReference(t.Context(), tenant, sale)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, domain.MovementSaleVoid, got[1].Type())
	})

	t.Run("la base de datos rechaza un sale_void sin referencia de venta", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `INSERT INTO stock_movements (id, tenant_id, variant_id, location_id, type, quantity,
				unit_cost, balance_after, average_cost_after, reason, user_id, occurred_at)
				VALUES ($1, $2, $3, $4, 'sale_void', 1, 0, 1, 0, '', $5, $6)`, newID(), tenant, variant, loc.ID(), newID(), now)
			return err
		})
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "stock_movements_sale_void_reference", pgErr.ConstraintName)
	})

	t.Run("aislamiento: otro tenant no ve los movimientos de la venta", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")
		sale := domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
		sell(t, tenant, variant, loc.ID(), sale, "3")

		got, err := f.stock.MovementsByReference(t.Context(), other, sale)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestStockLevelsByVariantAndLot(t *testing.T) {
	f := newFixture(t)

	t.Run("dos variantes del mismo producto tienen niveles y kardex separados", func(t *testing.T) {
		// Ropa: talla M y talla L de la misma camiseta.
		tenant := f.createTenant(t)
		variants := f.createProduct(t, tenant, 1)
		sizeM, sizeL := variants[0], variants[1]
		loc := f.createLocation(t, tenant, "Almacén")

		f.receive(t, tenant, sizeM, loc.ID(), "6", "15000")
		f.receive(t, tenant, sizeL, loc.ID(), "4", "16000")

		require.True(t, dec("6").Equal(f.level(t, tenant, sizeM, loc.ID())))
		require.True(t, dec("4").Equal(f.level(t, tenant, sizeL, loc.ID())))
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: sizeL, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, kardex.Total)
		require.Equal(t, sizeL, kardex.Items[0].VariantID())
	})

	t.Run("dos EnsureStockLevel concurrentes con lot_id NULL no crean dos niveles", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		const workers = 8
		start := make(chan struct{})
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				<-start
				errs[i] = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
					func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
						m, err := levels[0].Receive(dec("1"), dec("100"), info(newID()))
						return []*domain.StockMovement{m}, err
					})
			})
		}
		close(start)
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, 1, f.levelRows(t, tenant, variant, loc.ID()))
		require.True(t, dec("8").Equal(f.level(t, tenant, variant, loc.ID())))
	})

	t.Run("la base rechaza un segundo nivel con la misma variante, sede y lote NULL", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, variant, loc.ID(), "1", "100")

		err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
				VALUES ($1, $2, $3, NULL, 0, 0, $4)`, tenant, variant, loc.ID(), now)
			return err
		})
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "stock_levels_key", pgErr.ConstraintName)
	})

	t.Run("mover y listar el stock sin lote no toca ni muestra el nivel de un lote", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		lot := newID()
		f.exec(t, tenant, `INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
			VALUES ($1, $2, $3, $4, 5, 100, $5)`, tenant, variant, loc.ID(), lot, now)

		f.receive(t, tenant, variant, loc.ID(), "2", "100")

		require.Equal(t, 2, f.levelRows(t, tenant, variant, loc.ID()))
		var lotQty decimal.Decimal
		require.NoError(t, f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT quantity FROM stock_levels WHERE lot_id = $1", lot).Scan(&lotQty)
		}))
		require.True(t, dec("5").Equal(lotQty))
		page, err := f.stock.ListLevels(t.Context(), tenant, domain.StockFilter{LocationID: loc.ID(), Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total, "the core lists only the stock without a lot")
		require.True(t, dec("2").Equal(page.Items[0].Quantity()))
	})
}
