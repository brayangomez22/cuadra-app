package app_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// requireDec compares decimals by value, so 1100 equals 1100.00.
func requireDec(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	require.Truef(t, dec(want).Equal(got), "want %s, got %s", want, got)
}

func lineReq(variantID uuid.UUID, qty, discount string) app.LineRequest {
	return app.LineRequest{VariantID: variantID, Quantity: dec(qty), Discount: dec(discount)}
}

func cash(amount string) app.PaymentRequest {
	return app.PaymentRequest{Method: "cash", Amount: dec(amount)}
}

func TestNewService(t *testing.T) {
	t.Run("falla si falta un puerto", func(t *testing.T) {
		_, err := app.NewService(app.Deps{})
		require.Error(t, err)
	})
}

func TestConfirmSale(t *testing.T) {
	t.Run("confirma la venta y reserva el stock de cada línea", func(t *testing.T) {
		e := newEnv(t)
		cement := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
		cable := e.addProduct("Cable #12", "2500.50", "0.19", true)
		location := newID()

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: location,
			Lines: []app.LineRequest{
				lineReq(cement, "2", "0"),
				lineReq(cable, "2.5", "0"),
				lineReq(cement, "1", "0"),
			},
			Payments: []app.PaymentRequest{cash("120000")},
		})
		require.NoError(t, err)

		require.Equal(t, domain.SaleConfirmed, sale.Status())
		require.Equal(t, e.actor.TenantID, sale.TenantID())
		require.Equal(t, e.actor.UserID, sale.UserID())
		require.Equal(t, location, sale.LocationID())
		require.Equal(t, 7, int(sale.ID().Version()))
		require.Len(t, sale.Lines(), 3)
		// 90000 + 6251.25 = 96251.25 base; IVA 17100 + 1187.74 = 18287.74
		requireDec(t, "114538.99", sale.Totals().Total)
		requireDec(t, "5461.01", sale.Change())
		require.Equal(t, 1, e.storedSales())

		require.Len(t, e.store.reserved, 1)
		r := e.store.reserved[0]
		require.Equal(t, e.actor.TenantID, r.tenantID)
		require.Equal(t, e.actor.UserID, r.userID)
		require.Equal(t, location, r.locationID)
		require.Equal(t, sale.ID(), r.saleID)
		require.Len(t, r.lines, 2, "una línea de stock por variante")
		require.Equal(t, cement, r.lines[0].VariantID)
		requireDec(t, "3", r.lines[0].Quantity)
		require.Equal(t, cable, r.lines[1].VariantID)
		requireDec(t, "2.5", r.lines[1].Quantity)

		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.confirmed"))
		count, sum := e.histogram(t, "cuadra.sales.amount")
		require.Equal(t, uint64(1), count)
		require.InDelta(t, 114538.99, sum, 0.001)

		span, attrs := e.span(t, "sales.ConfirmSale")
		require.Equal(t, codes.Unset, span.Status().Code)
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, e.actor.UserID.String(), attrs["user.id"].AsString())
		require.Equal(t, location.String(), attrs["location.id"].AsString())
		require.Equal(t, sale.ID().String(), attrs["sale.id"].AsString())
	})

	t.Run("toma el nombre, el precio y el IVA del catálogo", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Tubo PVC 1/2", "8403.36", "0.19", true)
		e.addVariant(id, "3/4", "9000", true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{{Method: "nequi", Amount: dec("10000")}},
		})
		require.NoError(t, err)

		l := sale.Lines()[0]
		require.Equal(t, id, l.VariantID())
		require.Equal(t, e.productOf(id), l.ProductID())
		require.Equal(t, "Tubo PVC 1/2", l.ProductName())
		require.Empty(t, l.VariantLabel())
		require.Equal(t, domain.VariantPriceRule, l.PriceRule())
		requireDec(t, "8403.36", l.UnitPrice())
		requireDec(t, "0.19", l.TaxRate())
		requireDec(t, "10000", sale.Totals().Total)
		require.Len(t, e.store.catalogCalls, 1)
	})

	t.Run("pide al catálogo cada variante una sola vez", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Puntilla 2\"", "100", "0", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0"), lineReq(id, "2", "0")},
			Payments:   []app.PaymentRequest{cash("300")},
		})
		require.NoError(t, err)
		require.Equal(t, [][]uuid.UUID{{id}}, e.store.catalogCalls)
	})

	t.Run("el cajero no puede descontar por encima del tope del tenant", func(t *testing.T) {
		e := newEnv(t)
		e.setMaxDiscount(t, "0.10")
		id := e.addProduct("Pintura blanca galón", "50000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "5000.01")},
			Payments:   []app.PaymentRequest{cash("60000")},
		})
		require.ErrorIs(t, err, domain.ErrDiscountExceedsLimit)
		require.Zero(t, e.storedSales())
		require.Empty(t, e.store.reserved)
		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.discount_rejected"))
		require.Zero(t, e.counter(t, "cuadra.sales.confirmed"))
		span, _ := e.span(t, "sales.ConfirmSale")
		require.Equal(t, codes.Error, span.Status().Code)
	})

	t.Run("el cajero puede descontar hasta el tope del tenant", func(t *testing.T) {
		e := newEnv(t)
		e.setMaxDiscount(t, "0.10")
		id := e.addProduct("Pintura blanca galón", "50000", "0.19", true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "5000")},
			Payments:   []app.PaymentRequest{cash("60000")},
		})
		require.NoError(t, err)
		requireDec(t, "53550", sale.Totals().Total)
	})

	t.Run("sin tope configurado el cajero no puede descontar", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Lija #100", "2000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "1")},
			Payments:   []app.PaymentRequest{cash("5000")},
		})
		require.ErrorIs(t, err, domain.ErrDiscountExceedsLimit)
	})

	t.Run("quien tiene descuento libre no tiene tope", func(t *testing.T) {
		e := newEnv(t)
		e.setMaxDiscount(t, "0.10")
		e.actor.UnlimitedDiscount = true
		id := e.addProduct("Pintura blanca galón", "50000", "0.19", true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "25000")},
			Payments:   []app.PaymentRequest{cash("29750")},
		})
		require.NoError(t, err)
		requireDec(t, "29750", sale.Totals().Total)
	})

	t.Run("rechaza producto inexistente o de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		other := newEnv(t)
		foreign := other.addProduct("Producto ajeno", "1000", "0", true)

		for _, id := range []uuid.UUID{newID(), foreign} {
			_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
				LocationID: newID(),
				Lines:      []app.LineRequest{lineReq(id, "1", "0")},
				Payments:   []app.PaymentRequest{cash("1000")},
			})
			require.ErrorIs(t, err, domain.ErrProductNotFound)
		}
		require.Zero(t, e.storedSales())
	})

	t.Run("rechaza un producto o una variante inactivos", func(t *testing.T) {
		e := newEnv(t)
		inactiveProduct := e.addProduct("Descontinuado", "1000", "0", false)
		shirt := e.addProduct("Camiseta básica", "25000", "0.19", true)
		inactiveVariant := e.addVariant(shirt, "Verde · XL", "25000", false)

		for _, id := range []uuid.UUID{inactiveProduct, inactiveVariant} {
			_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
				LocationID: newID(),
				Lines:      []app.LineRequest{lineReq(id, "1", "0")},
				Payments:   []app.PaymentRequest{cash("30000")},
			})
			require.ErrorIs(t, err, domain.ErrProductInactive)
		}
		require.Zero(t, e.storedSales())
	})

	t.Run("rechaza pago insuficiente sin reservar stock", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Martillo", "20000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{cash("23799.99")},
		})
		require.ErrorIs(t, err, domain.ErrInsufficientPayment)
		require.Empty(t, e.store.reserved)
		require.Zero(t, e.storedSales())
	})

	t.Run("rechaza medio de pago desconocido", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Martillo", "20000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{{Method: "bitcoin", Amount: dec("23800")}},
		})
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
	})

	t.Run("rechaza una venta sin líneas", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{LocationID: newID()})
		require.ErrorIs(t, err, domain.ErrSaleWithoutLines)
		require.Empty(t, e.store.catalogCalls)
	})

	t.Run("si el StockReserver falla, la venta no queda guardada", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Martillo", "20000", "0.19", true)
		e.store.reserveErr = domain.ErrInsufficientStock

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{cash("23800")},
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		require.Zero(t, e.storedSales())
		require.Zero(t, e.counter(t, "cuadra.sales.confirmed"))
	})

	t.Run("propaga un error inesperado del StockReserver", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Martillo", "20000", "0.19", true)
		boom := errors.New("boom")
		e.store.reserveErr = boom

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{cash("23800")},
		})
		require.ErrorIs(t, err, boom)
	})

	t.Run("vender una variante reserva solo esa variante y guarda su etiqueta", func(t *testing.T) {
		e := newEnv(t)
		small := e.addProduct("Camiseta básica", "25000", "0.19", true)
		medium := e.addVariant(small, "Rojo · M", "26000", true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(medium, "1", "0")},
			Payments:   []app.PaymentRequest{cash("30940")},
		})
		require.NoError(t, err)

		l := sale.Lines()[0]
		require.Equal(t, medium, l.VariantID())
		require.Equal(t, e.productOf(small), l.ProductID())
		require.Equal(t, "Rojo · M", l.VariantLabel())
		requireDec(t, "26000", l.UnitPrice())
		require.Equal(t, []app.StockLine{{VariantID: medium, Quantity: dec("1")}}, e.store.reserved[0].lines)
	})

	t.Run("reserva por separado dos variantes del mismo producto y suma una variante repetida", func(t *testing.T) {
		tests := []struct {
			name, product, label string
		}{
			{"almacén de ropa: tallas de una camiseta", "Camiseta básica", "Rojo · M"},
			{"ferretería: presentaciones de una pintura", "Pintura blanca", "Cuarto"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				first := e.addProduct(tt.product, "20000", "0", true)
				second := e.addVariant(first, tt.label, "8000", true)

				_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
					LocationID: newID(),
					Lines: []app.LineRequest{
						lineReq(first, "1", "0"), lineReq(second, "2.5", "0"), lineReq(first, "1", "0"),
					},
					Payments: []app.PaymentRequest{cash("60000")},
				})
				require.NoError(t, err)

				lines := e.store.reserved[0].lines
				require.Len(t, lines, 2)
				require.Equal(t, first, lines[0].VariantID)
				requireDec(t, "2", lines[0].Quantity)
				require.Equal(t, second, lines[1].VariantID)
				requireDec(t, "2.5", lines[1].Quantity)
			})
		}
	})

	t.Run("product_id (deprecated) vende la variante por defecto del producto", func(t *testing.T) {
		e := newEnv(t)
		variant := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
		product := e.productOf(variant)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{{ProductID: product, Quantity: dec("1"), Discount: dec("0")}},
			Payments:   []app.PaymentRequest{cash("35700")},
		})
		require.NoError(t, err)
		require.Equal(t, variant, sale.Lines()[0].VariantID())
		require.Equal(t, product, sale.Lines()[0].ProductID())
		require.Equal(t, [][]uuid.UUID{{product}}, e.store.defaultCalls)
	})

	t.Run("product_id (deprecated) de un producto inexistente es product_not_found", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{{ProductID: newID(), Quantity: dec("1"), Discount: dec("0")}},
			Payments:   []app.PaymentRequest{cash("1000")},
		})
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("una línea lleva variant_id o product_id, no los dos ni ninguno", func(t *testing.T) {
		e := newEnv(t)
		variant := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)
		for _, l := range []app.LineRequest{
			{Quantity: dec("1"), Discount: dec("0")},
			{VariantID: variant, ProductID: e.productOf(variant), Quantity: dec("1"), Discount: dec("0")},
		} {
			_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
				LocationID: newID(), Lines: []app.LineRequest{l}, Payments: []app.PaymentRequest{cash("35700")},
			})
			require.ErrorIs(t, err, domain.ErrVariantRequired)
		}
		require.Empty(t, e.store.catalogCalls)
	})

	t.Run("resuelve los precios antes de abrir la transacción de la venta", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)

		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0"), lineReq(id, "2", "0")},
			Payments:   []app.PaymentRequest{cash("107100")},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"resolve", "resolve", "save"}, e.store.events)
		_, attrs := e.span(t, "sales.ResolvePrices")
		require.Equal(t, int64(2), attrs["sale.lines"].AsInt64())
		require.Equal(t, int64(2), e.counter(t, "cuadra.sales.price_rule_applied"))
	})
}
