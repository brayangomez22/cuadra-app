package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

var (
	strict     = domain.StockPolicy{}
	allowShort = domain.StockPolicy{AllowNegative: true}
)

func testInfo() domain.MovementInfo {
	return domain.MovementInfo{UserID: newID(), OccurredAt: testNow}
}

// requireDec compares decimals by value, so 1100 equals 1100.0000.
func requireDec(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	require.Truef(t, dec(want).Equal(got), "want %s, got %s", want, got)
}

func emptyLevel(t *testing.T) *domain.StockLevel {
	t.Helper()
	level, err := domain.NewStockLevel(newID(), newID(), newID())
	require.NoError(t, err)
	return level
}

func levelWith(t *testing.T, qty, avgCost string) *domain.StockLevel {
	t.Helper()
	level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
		TenantID:    newID(),
		ProductID:   newID(),
		LocationID:  newID(),
		Quantity:    dec(qty),
		AverageCost: dec(avgCost),
	})
	require.NoError(t, err)
	return level
}

func TestNewStockLevel(t *testing.T) {
	t.Run("crea nivel en cero para producto y sede", func(t *testing.T) {
		tenantID, productID, locationID := newID(), newID(), newID()
		level, err := domain.NewStockLevel(tenantID, productID, locationID)
		require.NoError(t, err)
		require.Equal(t, tenantID, level.TenantID())
		require.Equal(t, productID, level.ProductID())
		require.Equal(t, locationID, level.LocationID())
		requireDec(t, "0", level.Quantity())
		requireDec(t, "0", level.AverageCost())
	})

	t.Run("rechaza ids vacíos", func(t *testing.T) {
		_, err := domain.NewStockLevel(uuid.Nil, newID(), newID())
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
		_, err = domain.NewStockLevel(newID(), uuid.Nil, newID())
		require.ErrorIs(t, err, domain.ErrInvalidProductID)
		_, err = domain.NewStockLevel(newID(), newID(), uuid.Nil)
		require.ErrorIs(t, err, domain.ErrInvalidLocationID)
	})
}

func TestStockLevelReceive(t *testing.T) {
	t.Run("registra purchase_in con cantidad positiva y datos del nivel", func(t *testing.T) {
		level := emptyLevel(t)
		info := testInfo()
		ref := domain.DocumentRef{Kind: domain.DocumentPurchase, ID: newID()}
		info.Reference = &ref

		m, err := level.Receive(dec("10"), dec("1000"), info)
		require.NoError(t, err)
		require.Equal(t, 7, int(m.ID().Version()))
		require.Equal(t, level.TenantID(), m.TenantID())
		require.Equal(t, level.ProductID(), m.ProductID())
		require.Equal(t, level.LocationID(), m.LocationID())
		require.Equal(t, domain.MovementPurchaseIn, m.Type())
		requireDec(t, "10", m.Quantity())
		requireDec(t, "1000", m.UnitCost())
		require.Equal(t, &ref, m.Reference())
		require.Equal(t, info.UserID, m.UserID())
		require.Equal(t, testNow.UTC(), m.OccurredAt())
		require.Empty(t, m.Reason())
		requireDec(t, "10", level.Quantity())
		requireDec(t, "1000", level.AverageCost())
	})

	t.Run("costo promedio: 10 und a 1.000 + 10 und a 1.200 = 1.100", func(t *testing.T) {
		level := emptyLevel(t)
		_, err := level.Receive(dec("10"), dec("1000"), testInfo())
		require.NoError(t, err)
		_, err = level.Receive(dec("10"), dec("1200"), testInfo())
		require.NoError(t, err)

		requireDec(t, "20", level.Quantity())
		requireDec(t, "1100", level.AverageCost())
	})

	t.Run("costo promedio redondea half-up a 4 decimales", func(t *testing.T) {
		// (1 × 1 + 2 × 2) / 3 = 1.66666… → 1.6667
		level := emptyLevel(t)
		_, err := level.Receive(dec("1"), dec("1"), testInfo())
		require.NoError(t, err)
		_, err = level.Receive(dec("2"), dec("2"), testInfo())
		require.NoError(t, err)

		requireDec(t, "1.6667", level.AverageCost())
	})

	t.Run("una entrada con saldo negativo toma el costo de la entrada como promedio", func(t *testing.T) {
		level := levelWith(t, "-5", "900")
		_, err := level.Receive(dec("8"), dec("1000"), testInfo())
		require.NoError(t, err)

		requireDec(t, "3", level.Quantity())
		requireDec(t, "1000", level.AverageCost())
	})

	t.Run("cantidades fraccionarias (2,5 m) se suman sin error de precisión", func(t *testing.T) {
		level := emptyLevel(t)
		for _, q := range []string{"2.5", "0.1", "0.2"} {
			_, err := level.Receive(dec(q), dec("3200"), testInfo())
			require.NoError(t, err)
		}
		requireDec(t, "2.8", level.Quantity())
		require.Equal(t, "2.8", level.Quantity().String())
	})

	t.Run("rechaza cantidad cero, negativa o fuera de NUMERIC(18,4)", func(t *testing.T) {
		for _, q := range []string{"0", "-1", "1.00001", "100000000000000"} {
			level := emptyLevel(t)
			_, err := level.Receive(dec(q), dec("1000"), testInfo())
			require.ErrorIs(t, err, domain.ErrInvalidQuantity, q)
			requireDec(t, "0", level.Quantity())
		}
	})

	t.Run("rechaza una entrada que deja el saldo fuera de NUMERIC(18,4)", func(t *testing.T) {
		level := levelWith(t, "99999999999999", "1")
		_, err := level.Receive(dec("1"), dec("1"), testInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		requireDec(t, "99999999999999", level.Quantity())
	})

	t.Run("rechaza costo unitario negativo o fuera de NUMERIC(18,4)", func(t *testing.T) {
		for _, c := range []string{"-0.01", "1.00001", "100000000000000"} {
			level := emptyLevel(t)
			_, err := level.Receive(dec("1"), dec(c), testInfo())
			require.ErrorIs(t, err, domain.ErrInvalidUnitCost, c)
		}
	})

	t.Run("acepta costo unitario cero", func(t *testing.T) {
		level := emptyLevel(t)
		_, err := level.Receive(dec("1"), dec("0"), testInfo())
		require.NoError(t, err)
	})

	t.Run("rechaza movimiento sin usuario", func(t *testing.T) {
		level := emptyLevel(t)
		info := testInfo()
		info.UserID = uuid.Nil
		_, err := level.Receive(dec("1"), dec("1"), info)
		require.ErrorIs(t, err, domain.ErrInvalidUserID)
	})

	t.Run("rechaza referencia inválida", func(t *testing.T) {
		refs := map[string]domain.DocumentRef{
			"id vacío":           {Kind: domain.DocumentPurchase, ID: uuid.Nil},
			"tipo desconocido":   {Kind: "invoice", ID: newID()},
			"traslado reservado": {Kind: domain.DocumentTransfer, ID: newID()},
		}
		for name, ref := range refs {
			level := emptyLevel(t)
			info := testInfo()
			info.Reference = &ref
			_, err := level.Receive(dec("1"), dec("1"), info)
			require.ErrorIs(t, err, domain.ErrInvalidReference, name)
		}
	})
}

func TestStockLevelIssue(t *testing.T) {
	t.Run("una salida mayor al stock falla con ErrInsufficientStock y no cambia el nivel", func(t *testing.T) {
		level := levelWith(t, "5", "1000")
		_, err := level.Issue(dec("5.0001"), strict, testInfo())
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		requireDec(t, "5", level.Quantity())
	})

	t.Run("una salida por exactamente el stock deja el nivel en cero", func(t *testing.T) {
		level := levelWith(t, "5", "1000")
		_, err := level.Issue(dec("5"), strict, testInfo())
		require.NoError(t, err)
		requireDec(t, "0", level.Quantity())
	})

	t.Run("permite salida sin stock si la política del tenant admite negativos", func(t *testing.T) {
		level := levelWith(t, "1", "1000")
		_, err := level.Issue(dec("3"), allowShort, testInfo())
		require.NoError(t, err)
		requireDec(t, "-2", level.Quantity())
	})

	t.Run("la salida registra sale_out con cantidad negativa al costo promedio", func(t *testing.T) {
		level := levelWith(t, "10", "1100")
		info := testInfo()
		ref := domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
		info.Reference = &ref

		m, err := level.Issue(dec("2.5"), strict, info)
		require.NoError(t, err)
		require.Equal(t, domain.MovementSaleOut, m.Type())
		requireDec(t, "-2.5", m.Quantity())
		requireDec(t, "1100", m.UnitCost())
		require.Equal(t, &ref, m.Reference())
	})

	t.Run("una salida no cambia el costo promedio", func(t *testing.T) {
		level := levelWith(t, "10", "1100")
		_, err := level.Issue(dec("4"), strict, testInfo())
		require.NoError(t, err)
		requireDec(t, "1100", level.AverageCost())
	})

	t.Run("rechaza cantidad cero o negativa", func(t *testing.T) {
		for _, q := range []string{"0", "-1"} {
			level := levelWith(t, "10", "1")
			_, err := level.Issue(dec(q), strict, testInfo())
			require.ErrorIs(t, err, domain.ErrInvalidQuantity, q)
		}
	})
}

func TestStockLevelAdjust(t *testing.T) {
	t.Run("un ajuste exige motivo", func(t *testing.T) {
		for _, reason := range []string{"", "   "} {
			level := levelWith(t, "10", "1000")
			_, err := level.Adjust(dec("-1"), reason, strict, testInfo())
			require.ErrorIs(t, err, domain.ErrAdjustmentReasonRequired)
			requireDec(t, "10", level.Quantity())
		}
	})

	t.Run("rechaza motivo de más de 500 caracteres", func(t *testing.T) {
		level := levelWith(t, "10", "1000")
		_, err := level.Adjust(dec("-1"), strings.Repeat("a", 501), strict, testInfo())
		require.ErrorIs(t, err, domain.ErrAdjustmentReasonTooLong)
	})

	t.Run("registra adjustment con el delta y el motivo recortado", func(t *testing.T) {
		level := levelWith(t, "10", "1000")
		m, err := level.Adjust(dec("-1.5"), "  Bulto roto en bodega  ", strict, testInfo())
		require.NoError(t, err)
		require.Equal(t, domain.MovementAdjustment, m.Type())
		requireDec(t, "-1.5", m.Quantity())
		require.Equal(t, "Bulto roto en bodega", m.Reason())
		requireDec(t, "8.5", level.Quantity())
	})

	t.Run("un ajuste negativo mayor al stock falla si la política no admite negativos", func(t *testing.T) {
		level := levelWith(t, "2", "1000")
		_, err := level.Adjust(dec("-3"), "Conteo físico", strict, testInfo())
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		requireDec(t, "2", level.Quantity())

		_, err = level.Adjust(dec("-3"), "Conteo físico", allowShort, testInfo())
		require.NoError(t, err)
		requireDec(t, "-1", level.Quantity())
	})

	t.Run("un ajuste positivo entra al costo promedio y no lo cambia", func(t *testing.T) {
		level := levelWith(t, "10", "1100")
		m, err := level.Adjust(dec("2"), "Sobrante en conteo", strict, testInfo())
		require.NoError(t, err)
		requireDec(t, "1100", m.UnitCost())
		requireDec(t, "12", level.Quantity())
		requireDec(t, "1100", level.AverageCost())
	})

	t.Run("rechaza delta cero", func(t *testing.T) {
		level := levelWith(t, "10", "1000")
		_, err := level.Adjust(dec("0"), "Nada", strict, testInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})
}

func TestKardex(t *testing.T) {
	t.Run("el movimiento guarda saldo y costo promedio posteriores", func(t *testing.T) {
		level := emptyLevel(t)
		m1, err := level.Receive(dec("10"), dec("1000"), testInfo())
		require.NoError(t, err)
		m2, err := level.Receive(dec("10"), dec("1200"), testInfo())
		require.NoError(t, err)
		m3, err := level.Issue(dec("4"), strict, testInfo())
		require.NoError(t, err)

		requireDec(t, "10", m1.BalanceAfter())
		requireDec(t, "1000", m1.AverageCostAfter())
		requireDec(t, "20", m2.BalanceAfter())
		requireDec(t, "1100", m2.AverageCostAfter())
		requireDec(t, "16", m3.BalanceAfter())
		requireDec(t, "1100", m3.AverageCostAfter())
	})

	t.Run("la suma de los movimientos reconstruye el saldo del nivel", func(t *testing.T) {
		level := emptyLevel(t)
		other := siblingLevel(t, level, "0", "0")

		var movements []*domain.StockMovement
		add := func(m *domain.StockMovement, err error) {
			t.Helper()
			require.NoError(t, err)
			movements = append(movements, m)
		}
		add(level.Receive(dec("12.75"), dec("5000"), testInfo()))
		add(level.Issue(dec("2.5"), strict, testInfo()))
		add(level.Adjust(dec("-0.25"), "Merma", strict, testInfo()))
		out, _, err := domain.Transfer(level, other, dec("3"), strict, newID(), testNow)
		add(out, err)
		add(level.Receive(dec("1"), dec("5200"), testInfo()))

		sum := decimal.Zero
		for _, m := range movements {
			sum = sum.Add(m.Quantity())
		}
		requireDec(t, "8", sum)
		requireDec(t, "8", level.Quantity())
		requireDec(t, level.Quantity().String(), movements[len(movements)-1].BalanceAfter())
	})
}

func TestRehydrateStockLevel(t *testing.T) {
	t.Run("rechaza datos corruptos", func(t *testing.T) {
		valid := func() domain.StockLevelSnapshot {
			return domain.StockLevelSnapshot{
				TenantID: newID(), ProductID: newID(), LocationID: newID(),
				Quantity: dec("1"), AverageCost: dec("1"),
			}
		}
		cases := map[string]struct {
			mutate func(*domain.StockLevelSnapshot)
			want   error
		}{
			"tenant vacío":             {func(s *domain.StockLevelSnapshot) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			"producto vacío":           {func(s *domain.StockLevelSnapshot) { s.ProductID = uuid.Nil }, domain.ErrInvalidProductID},
			"sede vacía":               {func(s *domain.StockLevelSnapshot) { s.LocationID = uuid.Nil }, domain.ErrInvalidLocationID},
			"cantidad con 5 decimales": {func(s *domain.StockLevelSnapshot) { s.Quantity = dec("1.00001") }, domain.ErrInvalidStockLevel},
			"costo promedio negativo":  {func(s *domain.StockLevelSnapshot) { s.AverageCost = dec("-1") }, domain.ErrInvalidStockLevel},
		}
		for name, tc := range cases {
			s := valid()
			tc.mutate(&s)
			_, err := domain.RehydrateStockLevel(s)
			require.ErrorIs(t, err, tc.want, name)
		}
	})

	t.Run("acepta saldo negativo guardado", func(t *testing.T) {
		level := levelWith(t, "-2", "1000")
		requireDec(t, "-2", level.Quantity())
	})
}
