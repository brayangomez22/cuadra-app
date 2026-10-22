package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

type variantUnitJSON struct {
	Unit struct {
		ID, Code, Name string
	} `json:"unit"`
	Factor       string  `json:"factor"`
	Price        string  `json:"price"`
	PriceWithTax string  `json:"price_with_tax"`
	Barcode      *string `json:"barcode"`
}

// unitsOf decodes the alternate units of each variant of a product.
func unitsOf(t *testing.T, raw []byte) [][]variantUnitJSON {
	t.Helper()
	var body struct {
		Variants []struct {
			Units []variantUnitJSON `json:"units"`
		} `json:"variants"`
	}
	require.NoError(t, json.Unmarshal(raw, &body), string(raw))
	out := make([][]variantUnitJSON, 0, len(body.Variants))
	for _, v := range body.Variants {
		require.NotNil(t, v.Units, "units siempre viene, vacío si no hay")
		out = append(out, v.Units)
	}
	return out
}

func TestVariantUnitsBody(t *testing.T) {
	tests := []struct {
		name                       string
		product                    func(t *testing.T) *domain.Product
		code, unitName             string
		factor, price, priceWithTx string
	}{
		{"ferretería: caja de 12 de un producto por metro", func(t *testing.T) *domain.Product { return product(t, "8000", "10000") }, "caja", "Caja", "12", "110000", "130900.00"},
		{"almacén de ropa: paquete de 3 camisetas", productWithVariants, "paquete", "Paquete", "3", "70000.5", "83300.60"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.product(t)
			unit := mustUnit(tt.code, tt.unitName)
			require.NoError(t, p.SetVariantUnit(p.DefaultVariant().ID(), domain.VariantUnitParams{
				Unit: unit.Ref(), Factor: decimal.RequireFromString(tt.factor), Price: decimal.RequireFromString(tt.price),
			}))

			rec := serve(t, &stubUseCases{product: p}, cashier(), http.MethodGet, "/api/v1/products/"+p.ID().String(), "")

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			units := unitsOf(t, rec.Body.Bytes())
			require.Len(t, units[0], 1)
			got := units[0][0]
			require.Equal(t, unit.ID().String(), got.Unit.ID)
			require.Equal(t, tt.code, got.Unit.Code)
			require.Equal(t, tt.factor, got.Factor)
			require.True(t, decimal.RequireFromString(tt.price).Equal(decimal.RequireFromString(got.Price)))
			require.Equal(t, tt.priceWithTx, got.PriceWithTax)
			require.Nil(t, got.Barcode)
			for _, other := range units[1:] {
				require.Empty(t, other)
			}
		})
	}
}

func TestSetVariantUnit(t *testing.T) {
	path := func(p *domain.Product, unit *domain.UnitOfMeasure) string {
		return "/api/v1/products/" + p.ID().String() + "/variants/" + p.DefaultVariant().ID().String() + "/units/" + unit.ID().String()
	}

	t.Run("agrega la unidad y responde el producto", func(t *testing.T) {
		p := product(t, "8000", "10000")
		box := mustUnit("caja", "Caja")
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, owner, http.MethodPut, path(p, box), `{"factor":"12","price":"110000","barcode":"7703333333333"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.ID(), uc.id)
		require.Equal(t, p.DefaultVariant().ID(), uc.variantID)
		require.Equal(t, box.ID(), uc.unitID)
		require.True(t, decimal.RequireFromString("12").Equal(uc.unitInput.Factor))
		require.True(t, decimal.RequireFromString("110000").Equal(uc.unitInput.Price))
		require.Equal(t, "7703333333333", uc.unitInput.Barcode)
	})

	t.Run("traduce los errores a su código", func(t *testing.T) {
		p := product(t, "8000", "10000")
		box := mustUnit("caja", "Caja")
		tests := []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{"unidad inexistente", domain.ErrUnitNotFound, http.StatusNotFound, "unit_not_found"},
			{"unidad base", domain.ErrAlternateUnitIsBase, http.StatusUnprocessableEntity, "alternate_unit_is_base"},
			{"factor inválido", domain.ErrInvalidUnitFactor, http.StatusUnprocessableEntity, "invalid_unit_factor"},
			{"código de barras repetido", domain.ErrBarcodeTaken, http.StatusConflict, "barcode_taken"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rec := serve(t, &stubUseCases{err: tt.err}, owner, http.MethodPut, path(p, box), `{"factor":"12","price":"110000"}`)
				requireError(t, rec, tt.status, tt.code)
			})
		}
	})

	t.Run("un factor que no es un número es invalid_unit_factor sin llamar al caso de uso", func(t *testing.T) {
		p := product(t, "8000", "10000")
		uc := &stubUseCases{product: p}
		rec := serve(t, uc, owner, http.MethodPut, path(p, mustUnit("caja", "Caja")), `{"factor":"doce","price":"110000"}`)
		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_unit_factor")
		require.Zero(t, uc.unitID)
	})

	t.Run("quita la unidad; si la variante no la tiene responde 404", func(t *testing.T) {
		p := product(t, "8000", "10000")
		box := mustUnit("caja", "Caja")
		uc := &stubUseCases{product: p}

		rec := serve(t, uc, owner, http.MethodDelete, path(p, box), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, box.ID(), uc.unitID)

		rec = serve(t, &stubUseCases{err: domain.ErrVariantUnitNotFound}, owner, http.MethodDelete, path(p, box), "")
		requireError(t, rec, http.StatusNotFound, "variant_unit_not_found")
	})
}
