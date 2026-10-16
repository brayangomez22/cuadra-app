package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// productWithAxes returns a product whose default variant has axes, so more
// variants can be added: a garment by size and color, or a paint by size of
// the can.
func productWithAxes(t *testing.T, axes map[string]string) *domain.Product {
	t.Helper()
	p := validProductParams(t)
	p.Variant.AxisValues = axes
	product, err := domain.NewProduct(p, testNow)
	require.NoError(t, err)
	return product
}

func variantParams(sku string, axes map[string]string) domain.VariantParams {
	return domain.VariantParams{SKU: sku, Cost: dec("15000"), Price: dec("30000"), AxisValues: axes}
}

func TestProductAddVariant(t *testing.T) {
	tests := []struct {
		name          string
		defaultAxes   map[string]string
		newAxes       map[string]string
		duplicateAxes map[string]string
		otherAxesKeys map[string]string
	}{
		{
			name:          "almacén de ropa: talla y color",
			defaultAxes:   map[string]string{"talla": "S", "color": "Rojo"},
			newAxes:       map[string]string{"talla": "M", "color": "Rojo"},
			duplicateAxes: map[string]string{"color": "Rojo", "talla": "S"},
			otherAxesKeys: map[string]string{"talla": "M"},
		},
		{
			name:          "ferretería: presentación de una pintura",
			defaultAxes:   map[string]string{"presentación": "Galón"},
			newAxes:       map[string]string{"presentación": "Cuarto"},
			duplicateAxes: map[string]string{"presentación": "Galón"},
			otherAxesKeys: map[string]string{"color": "Blanco"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("agrega una variante activa con su propio id", func(t *testing.T) {
				product := productWithAxes(t, tt.defaultAxes)

				v, err := product.AddVariant(variantParams(" var-2 ", tt.newAxes), testNow)
				require.NoError(t, err)
				require.Equal(t, 7, int(v.ID().Version()))
				require.NotEqual(t, product.DefaultVariant().ID(), v.ID())
				require.Equal(t, product.ID(), v.ProductID())
				require.Equal(t, "VAR-2", v.SKU())
				require.Equal(t, tt.newAxes, v.AxisValues())
				require.False(t, v.IsDefault())
				require.True(t, v.IsActive())
				require.Len(t, product.Variants(), 2)

				got, err := product.Variant(v.ID())
				require.NoError(t, err)
				require.Same(t, v, got)
			})

			t.Run("rechaza dos variantes con los mismos valores de ejes", func(t *testing.T) {
				product := productWithAxes(t, tt.defaultAxes)

				_, err := product.AddVariant(variantParams("VAR-2", tt.duplicateAxes), testNow)
				require.ErrorIs(t, err, domain.ErrDuplicateVariant)
				require.Len(t, product.Variants(), 1)
			})

			t.Run("rechaza una variante con ejes distintos a los de las demás", func(t *testing.T) {
				product := productWithAxes(t, tt.defaultAxes)

				_, err := product.AddVariant(variantParams("VAR-2", tt.otherAxesKeys), testNow)
				require.ErrorIs(t, err, domain.ErrVariantAxesMismatch)
				require.Len(t, product.Variants(), 1)
			})
		})
	}

	t.Run("a un producto simple no se le agrega otra variante sin ejes", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")

		_, err := product.AddVariant(variantParams("OTRA", nil), testNow)
		require.ErrorIs(t, err, domain.ErrDuplicateVariant)
	})

	t.Run("a un producto simple no se le agrega una variante con ejes", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")

		_, err := product.AddVariant(variantParams("OTRA", map[string]string{"talla": "M"}), testNow)
		require.ErrorIs(t, err, domain.ErrVariantAxesMismatch)
	})

	t.Run("rechaza un SKU o un código de barras que ya usa otra variante del producto", func(t *testing.T) {
		product := productWithAxes(t, map[string]string{"talla": "S"})

		_, err := product.AddVariant(variantParams("cem-gris-50", map[string]string{"talla": "M"}), testNow)
		require.ErrorIs(t, err, domain.ErrSKUTaken)

		withBarcode := variantParams("VAR-2", map[string]string{"talla": "M"})
		withBarcode.Barcode = "7701234567890"
		_, err = product.AddVariant(withBarcode, testNow)
		require.ErrorIs(t, err, domain.ErrBarcodeTaken)
		require.Len(t, product.Variants(), 1)
	})

	t.Run("rechaza una variante con datos inválidos", func(t *testing.T) {
		axes := map[string]string{"talla": "M"}
		tests := []struct {
			name   string
			mutate func(*domain.VariantParams)
			want   error
		}{
			{"SKU vacío", func(v *domain.VariantParams) { v.SKU = "  " }, domain.ErrInvalidSKU},
			{"SKU con espacios internos", func(v *domain.VariantParams) { v.SKU = "CAM M" }, domain.ErrInvalidSKU},
			{"SKU demasiado largo", func(v *domain.VariantParams) { v.SKU = strings.Repeat("A", 65) }, domain.ErrInvalidSKU},
			{"código de barras con espacios", func(v *domain.VariantParams) { v.Barcode = "770 123" }, domain.ErrInvalidBarcode},
			{"código de barras no ASCII", func(v *domain.VariantParams) { v.Barcode = "77012ñ" }, domain.ErrInvalidBarcode},
			{"precio cero", func(v *domain.VariantParams) { v.Price = dec("0") }, domain.ErrInvalidPrice},
			{"precio negativo", func(v *domain.VariantParams) { v.Price = dec("-100") }, domain.ErrInvalidPrice},
			{"precio con más de 4 decimales", func(v *domain.VariantParams) { v.Price = dec("100.12345") }, domain.ErrInvalidPrice},
			{"precio fuera de rango", func(v *domain.VariantParams) { v.Price = dec("100000000000000") }, domain.ErrInvalidPrice},
			{"costo negativo", func(v *domain.VariantParams) { v.Cost = dec("-0.01") }, domain.ErrInvalidCost},
			{"costo con más de 4 decimales", func(v *domain.VariantParams) { v.Cost = dec("0.00001") }, domain.ErrInvalidCost},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				product := productWithAxes(t, map[string]string{"talla": "S"})
				v := variantParams("CAM-M", axes)
				tt.mutate(&v)

				_, err := product.AddVariant(v, testNow)
				require.ErrorIs(t, err, tt.want)
				require.Len(t, product.Variants(), 1)
			})
		}
	})
}

func TestAxisValues(t *testing.T) {
	t.Run("normaliza los ejes: claves en minúsculas y espacios recortados", func(t *testing.T) {
		//nolint:gocritic // the spaces and the case are what normalizeAxes removes
		product := productWithAxes(t, map[string]string{"  Talla ": " M ", "COLOR": "Azul marino "})

		require.Equal(t, map[string]string{"talla": "M", "color": "Azul marino"}, product.DefaultVariant().AxisValues())
	})

	t.Run("rechaza ejes inválidos", func(t *testing.T) {
		tests := []struct {
			name string
			axes map[string]string
		}{
			{"clave vacía", map[string]string{" ": "M"}},
			{"valor vacío", map[string]string{"talla": "  "}},
			{"clave demasiado larga", map[string]string{strings.Repeat("a", 51): "M"}},
			{"valor demasiado largo", map[string]string{"talla": strings.Repeat("a", 101)}},
			{"dos claves iguales al normalizar", map[string]string{"talla": "M", " TALLA": "L"}}, //nolint:gocritic // the collision under test
			{"más de 5 ejes", map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := validProductParams(t)
				p.Variant.AxisValues = tt.axes

				_, err := domain.NewProduct(p, testNow)
				require.ErrorIs(t, err, domain.ErrInvalidAxisValues)
			})
		}
	})

	t.Run("los ejes devueltos son una copia", func(t *testing.T) {
		product := productWithAxes(t, map[string]string{"talla": "S"})

		product.DefaultVariant().AxisValues()["talla"] = "XL"
		require.Equal(t, "S", product.DefaultVariant().AxisValues()["talla"])
	})
}

func TestProductUpdateVariant(t *testing.T) {
	t.Run("actualiza SKU, código, costo y ejes sin cambiar el precio", func(t *testing.T) {
		product := productWithAxes(t, map[string]string{"talla": "S"})
		v := product.DefaultVariant()

		err := product.UpdateVariant(v.ID(), domain.VariantDetails{
			SKU: " cam-s ", Barcode: " 7709876543210 ", Cost: dec("3500.1234"),
			AxisValues: map[string]string{"talla": "XS"},
		})
		require.NoError(t, err)

		v = product.DefaultVariant()
		require.Equal(t, "CAM-S", v.SKU())
		require.Equal(t, "7709876543210", v.Barcode())
		require.True(t, dec("3500.1234").Equal(v.Cost()))
		require.Equal(t, map[string]string{"talla": "XS"}, v.AxisValues())
		require.True(t, dec("32500.50").Equal(v.Price()), "the price changes only through ChangeVariantPrice")
		require.True(t, v.IsDefault())
	})

	t.Run("la única variante puede pasar de no tener ejes a tenerlos", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		id := product.DefaultVariant().ID()

		require.NoError(t, product.UpdateVariant(id, domain.VariantDetails{
			SKU: "CAM-S", Cost: dec("8000"), AxisValues: map[string]string{"talla": "S"},
		}))
		_, err := product.AddVariant(variantParams("CAM-M", map[string]string{"talla": "M"}), testNow)
		require.NoError(t, err)
	})

	t.Run("rechaza una actualización inválida sin modificar la variante", func(t *testing.T) {
		tests := []struct {
			name string
			d    domain.VariantDetails
			want error
		}{
			{"SKU vacío", domain.VariantDetails{SKU: " ", Cost: dec("1"), AxisValues: map[string]string{"talla": "S"}}, domain.ErrInvalidSKU},
			{"costo negativo", domain.VariantDetails{SKU: "A", Cost: dec("-1"), AxisValues: map[string]string{"talla": "S"}}, domain.ErrInvalidCost},
			{"ejes de otra variante", domain.VariantDetails{SKU: "A", Cost: dec("1"), AxisValues: map[string]string{"talla": "M"}}, domain.ErrDuplicateVariant},
			{"ejes con otras claves", domain.VariantDetails{SKU: "A", Cost: dec("1"), AxisValues: map[string]string{"color": "Rojo"}}, domain.ErrVariantAxesMismatch},
			{"SKU de otra variante", domain.VariantDetails{SKU: "cam-m", Cost: dec("1"), AxisValues: map[string]string{"talla": "S"}}, domain.ErrSKUTaken},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				product := productWithAxes(t, map[string]string{"talla": "S"})
				_, err := product.AddVariant(variantParams("CAM-M", map[string]string{"talla": "M"}), testNow)
				require.NoError(t, err)
				before := *product.DefaultVariant()

				require.ErrorIs(t, product.UpdateVariant(before.ID(), tt.d), tt.want)
				require.Equal(t, before, *product.DefaultVariant())
			})
		}
	})

	t.Run("una variante que no es del producto no se encuentra", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		missing := uuid.Must(uuid.NewV7())

		_, err := product.Variant(missing)
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
		require.ErrorIs(t, product.UpdateVariant(missing, domain.VariantDetails{SKU: "A", Cost: dec("1")}), domain.ErrVariantNotFound)
		require.ErrorIs(t, product.ChangeVariantPrice(missing, dec("1")), domain.ErrVariantNotFound)
		require.ErrorIs(t, product.SetVariantActive(missing, false), domain.ErrVariantNotFound)
	})
}

func TestProductChangeVariantPrice(t *testing.T) {
	t.Run("cambia el precio de una variante de un producto activo", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		id := product.DefaultVariant().ID()

		require.NoError(t, product.ChangeVariantPrice(id, dec("11500.25")))
		require.True(t, dec("11500.25").Equal(product.DefaultVariant().Price()))
	})

	t.Run("rechaza cambio a precio negativo o cero", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		id := product.DefaultVariant().ID()
		for _, price := range []string{"0", "-1", "10.00001"} {
			require.ErrorIs(t, product.ChangeVariantPrice(id, dec(price)), domain.ErrInvalidPrice)
		}
		require.True(t, dec("10000").Equal(product.DefaultVariant().Price()))
	})

	t.Run("una variante de un producto inactivo no cambia de precio", func(t *testing.T) {
		product := mustProduct(t, "8000", "10000")
		product.Deactivate()

		require.ErrorIs(t, product.ChangeVariantPrice(product.DefaultVariant().ID(), dec("12000")), domain.ErrProductInactive)
		require.True(t, dec("10000").Equal(product.DefaultVariant().Price()))
	})
}

func TestProductSetVariantActive(t *testing.T) {
	t.Run("desactiva y reactiva una variante sin tocar el producto", func(t *testing.T) {
		product := productWithAxes(t, map[string]string{"talla": "S"})
		v, err := product.AddVariant(variantParams("CAM-M", map[string]string{"talla": "M"}), testNow)
		require.NoError(t, err)

		require.NoError(t, product.SetVariantActive(v.ID(), false))
		got, err := product.Variant(v.ID())
		require.NoError(t, err)
		require.False(t, got.IsActive())
		require.True(t, product.IsActive())

		require.NoError(t, product.SetVariantActive(v.ID(), true))
		got, err = product.Variant(v.ID())
		require.NoError(t, err)
		require.True(t, got.IsActive())
	})
}

func TestVariantMargin(t *testing.T) {
	tests := []struct {
		name, cost, price, want string
	}{
		{"calcula el margen sin pérdida de precisión", "8333.33", "10000", "16.6667"},
		// (3 - 1) / 3 × 100 = 66.66666…
		{"redondea el margen half-up a 4 decimales", "1", "3", "66.6667"},
		{"margen es 100 cuando el costo es cero", "0", "5000", "100"},
		{"margen es negativo cuando se vende por debajo del costo", "12000", "10000", "-20"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := mustProduct(t, tt.cost, tt.price)
			require.True(t, dec(tt.want).Equal(product.DefaultVariant().MarginPercent()),
				"got %s", product.DefaultVariant().MarginPercent())
		})
	}
}

func TestVariantPriceWithTax(t *testing.T) {
	tests := []struct {
		name, price, rate, want string
	}{
		{"suma el IVA del 19% y redondea a 2 decimales", "8403.36", "0.19", "10000.00"},
		{"redondea half-up", "10.0250", "0", "10.03"},
		{"suma el IVA del 5%", "1000", "0.05", "1050.00"},
		{"variante exenta queda igual", "2500", "0", "2500.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := mustProduct(t, "0", tt.price)
			got := product.DefaultVariant().PriceWithTax(mustTaxRate(t, tt.rate))
			require.Equal(t, tt.want, got.StringFixed(2))
		})
	}
}

func TestVariantLabel(t *testing.T) {
	tests := []struct {
		name string
		axes map[string]string
		want string
	}{
		{"producto simple de una ferretería: sin ejes no hay etiqueta", nil, ""},
		{"ferretería: pintura por presentación", map[string]string{"presentación": "Galón"}, "Galón"},
		{"almacén de ropa: los valores van ordenados por nombre de eje", map[string]string{"talla": "M", "color": "Rojo"}, "Rojo · M"},
		{"almacén de calzado: tres ejes", map[string]string{"talla": "38", "color": "Negro", "ancho": "E"}, "E · Negro · 38"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, productWithAxes(t, tt.axes).DefaultVariant().Label())
		})
	}
}
