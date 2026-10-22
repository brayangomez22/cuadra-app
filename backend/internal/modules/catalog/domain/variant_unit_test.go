package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// Alternate units of the tenants in the tests.
var (
	roll   = unitRef("rollo", "Rollo")
	box    = unitRef("caja", "Caja")
	pack   = unitRef("paquete", "Paquete")
	grams  = unitRef("g", "Gramo")
	kilos  = unitRef("kg", "Kilogramo")
	dozens = unitRef("docena", "Docena")
)

// productIn returns a product counted in base, with one more variant when
// second is true.
func productIn(t *testing.T, base domain.UnitRef, second bool) *domain.Product {
	t.Helper()
	p := validProductParams(t)
	p.BaseUnit = base
	if second {
		p.Variant.AxisValues = map[string]string{"color": "Negro"}
	}
	product, err := domain.NewProduct(p, testNow)
	require.NoError(t, err)
	if second {
		_, err = product.AddVariant(domain.VariantParams{
			SKU: "VAR-2", Barcode: "7700000000002", Cost: dec("1000"), Price: dec("2000"),
			AxisValues: map[string]string{"color": "Blanco"},
		}, testNow)
		require.NoError(t, err)
	}
	return product
}

func TestProductSetVariantUnit(t *testing.T) {
	tests := []struct {
		name      string
		base      domain.UnitRef
		alternate domain.VariantUnitParams
		other     domain.UnitRef
	}{
		{
			name:      "ferretería: rollo de 100 m de un cable que se vende por metro",
			base:      meters,
			alternate: domain.VariantUnitParams{Unit: roll, Factor: dec("100"), Price: dec("180000"), Barcode: "7701111111111"},
			other:     box,
		},
		{
			name:      "almacén de ropa: paquete de 3 camisetas que se venden por unidad",
			base:      piece,
			alternate: domain.VariantUnitParams{Unit: pack, Factor: dec("3"), Price: dec("75000"), Barcode: "7702222222222"},
			other:     dozens,
		},
		{
			name:      "tienda: gramos de un producto que se cuenta por kilo",
			base:      kilos,
			alternate: domain.VariantUnitParams{Unit: grams, Factor: dec("0.001"), Price: dec("12.5")},
			other:     box,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("agrega una unidad alternativa con su factor, precio y código de barras", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()

				require.NoError(t, p.SetVariantUnit(v.ID(), tt.alternate))

				units := p.DefaultVariant().Units()
				require.Len(t, units, 1)
				require.Equal(t, tt.alternate.Unit, units[0].Unit())
				require.True(t, tt.alternate.Factor.Equal(units[0].Factor()))
				require.True(t, tt.alternate.Price.Equal(units[0].Price()))
				require.Equal(t, tt.alternate.Barcode, units[0].Barcode())

				got, err := p.DefaultVariant().Unit(tt.alternate.Unit.ID)
				require.NoError(t, err)
				require.Equal(t, units[0], got)
			})

			t.Run("reemplaza la unidad si la variante ya la tiene", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()
				require.NoError(t, p.SetVariantUnit(v.ID(), tt.alternate))

				changed := tt.alternate
				changed.Price = changed.Price.Add(dec("1"))
				changed.Barcode = ""
				require.NoError(t, p.SetVariantUnit(v.ID(), changed))

				units := p.DefaultVariant().Units()
				require.Len(t, units, 1)
				require.True(t, changed.Price.Equal(units[0].Price()))
				require.Empty(t, units[0].Barcode())
			})

			t.Run("ordena las unidades por factor", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()
				bigger := domain.VariantUnitParams{Unit: tt.other, Factor: tt.alternate.Factor.Mul(dec("10")), Price: tt.alternate.Price.Mul(dec("9"))}
				require.NoError(t, p.SetVariantUnit(v.ID(), bigger))
				require.NoError(t, p.SetVariantUnit(v.ID(), tt.alternate))

				units := p.DefaultVariant().Units()
				require.Len(t, units, 2)
				require.Equal(t, tt.alternate.Unit, units[0].Unit())
				require.Equal(t, tt.other, units[1].Unit())
			})

			t.Run("rechaza datos inválidos y deja el producto como estaba", func(t *testing.T) {
				p := productIn(t, tt.base, true)
				v := p.DefaultVariant()
				other := p.Variants()[1]
				cases := []struct {
					name   string
					change func(*domain.VariantUnitParams)
					want   error
				}{
					{"la unidad base del producto", func(u *domain.VariantUnitParams) { u.Unit = tt.base }, domain.ErrAlternateUnitIsBase},
					{"sin unidad", func(u *domain.VariantUnitParams) { u.Unit = domain.UnitRef{} }, domain.ErrInvalidUnit},
					{"factor cero", func(u *domain.VariantUnitParams) { u.Factor = dec("0") }, domain.ErrInvalidUnitFactor},
					{"factor negativo", func(u *domain.VariantUnitParams) { u.Factor = dec("-12") }, domain.ErrInvalidUnitFactor},
					{"factor con más de 4 decimales", func(u *domain.VariantUnitParams) { u.Factor = dec("0.00001") }, domain.ErrInvalidUnitFactor},
					{"precio cero", func(u *domain.VariantUnitParams) { u.Price = dec("0") }, domain.ErrInvalidPrice},
					{"código de barras con espacios", func(u *domain.VariantUnitParams) { u.Barcode = "77 01" }, domain.ErrInvalidBarcode},
					{"código de barras de una variante del producto", func(u *domain.VariantUnitParams) { u.Barcode = other.Barcode() }, domain.ErrBarcodeTaken},
				}
				for _, c := range cases {
					t.Run(c.name, func(t *testing.T) {
						params := tt.alternate
						c.change(&params)
						err := p.SetVariantUnit(v.ID(), params)
						require.ErrorIs(t, err, c.want)
						require.Empty(t, p.DefaultVariant().Units())
					})
				}
			})

			t.Run("rechaza el código de barras de una unidad de otra variante del producto", func(t *testing.T) {
				p := productIn(t, tt.base, true)
				withBarcode := tt.alternate
				withBarcode.Barcode = "7709999999999"
				require.NoError(t, p.SetVariantUnit(p.DefaultVariant().ID(), withBarcode))

				err := p.SetVariantUnit(p.Variants()[1].ID(), withBarcode)
				require.ErrorIs(t, err, domain.ErrBarcodeTaken)
				require.Empty(t, p.Variants()[1].Units())
			})

			t.Run("rechaza más de 10 unidades alternativas en una variante", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()
				for i := range 10 {
					extra := domain.VariantUnitParams{Unit: unitRef("u"+string(rune('a'+i)), "Extra"), Factor: dec("2"), Price: dec("10")}
					require.NoError(t, p.SetVariantUnit(v.ID(), extra))
				}
				err := p.SetVariantUnit(v.ID(), tt.alternate)
				require.ErrorIs(t, err, domain.ErrTooManyVariantUnits)
				require.Len(t, p.DefaultVariant().Units(), 10)
			})

			t.Run("rechaza una variante que no es del producto", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				err := p.SetVariantUnit(uuid.Must(uuid.NewV7()), tt.alternate)
				require.ErrorIs(t, err, domain.ErrVariantNotFound)
			})

			t.Run("quita una unidad alternativa", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()
				require.NoError(t, p.SetVariantUnit(v.ID(), tt.alternate))

				require.NoError(t, p.RemoveVariantUnit(v.ID(), tt.alternate.Unit.ID))
				require.Empty(t, p.DefaultVariant().Units())

				_, err := p.DefaultVariant().Unit(tt.alternate.Unit.ID)
				require.ErrorIs(t, err, domain.ErrVariantUnitNotFound)
			})

			t.Run("rechaza quitar una unidad que la variante no tiene", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				err := p.RemoveVariantUnit(p.DefaultVariant().ID(), tt.alternate.Unit.ID)
				require.ErrorIs(t, err, domain.ErrVariantUnitNotFound)
			})

			t.Run("rechaza cambiar la unidad base a una unidad alternativa de una variante", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				require.NoError(t, p.SetVariantUnit(p.DefaultVariant().ID(), tt.alternate))

				err := p.UpdateDetails(domain.ProductDetails{
					Name: p.Name(), BaseUnit: tt.alternate.Unit, TaxRate: p.TaxRate(), Attributes: p.Attributes(),
				})
				require.ErrorIs(t, err, domain.ErrAlternateUnitIsBase)
				require.Equal(t, tt.base, p.BaseUnit())
			})

			t.Run("conserva las unidades al editar la variante", func(t *testing.T) {
				p := productIn(t, tt.base, false)
				v := p.DefaultVariant()
				require.NoError(t, p.SetVariantUnit(v.ID(), tt.alternate))

				require.NoError(t, p.UpdateVariant(v.ID(), domain.VariantDetails{SKU: "OTRO-SKU", Cost: v.Cost()}))
				require.NoError(t, p.ChangeVariantPrice(v.ID(), v.Price().Add(dec("1"))))
				require.NoError(t, p.SetVariantActive(v.ID(), false))

				require.Len(t, p.DefaultVariant().Units(), 1)
			})
		})
	}
}

func TestRehydrateProductWithVariantUnits(t *testing.T) {
	snapshot := func(units ...domain.VariantUnitParams) domain.ProductSnapshot {
		return domain.ProductSnapshot{
			ID: uuid.Must(uuid.NewV7()), TenantID: uuid.Must(uuid.NewV7()), Name: "Cable", BaseUnit: meters,
			TaxRate: mustTaxRate(t, "0.19"), Active: true, CreatedAt: testNow,
			Variants: []domain.VariantSnapshot{{
				ID: uuid.Must(uuid.NewV7()), SKU: "CAB-1", Cost: dec("1000"), Price: dec("2000"),
				IsDefault: true, Active: true, CreatedAt: testNow, Units: units,
			}},
		}
	}
	rollUnit := domain.VariantUnitParams{Unit: roll, Factor: dec("100"), Price: dec("180000")}

	t.Run("rehidrata un producto con unidades alternativas", func(t *testing.T) {
		p, err := domain.RehydrateProduct(snapshot(rollUnit))
		require.NoError(t, err)
		require.Len(t, p.DefaultVariant().Units(), 1)
	})

	t.Run("rechaza datos guardados con una unidad repetida en la variante", func(t *testing.T) {
		_, err := domain.RehydrateProduct(snapshot(rollUnit, rollUnit))
		require.ErrorIs(t, err, domain.ErrDuplicateVariantUnit)
	})

	t.Run("rechaza datos guardados con la unidad base como alternativa", func(t *testing.T) {
		_, err := domain.RehydrateProduct(snapshot(domain.VariantUnitParams{Unit: meters, Factor: dec("1"), Price: dec("2000")}))
		require.ErrorIs(t, err, domain.ErrAlternateUnitIsBase)
	})
}
