package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
)

// setDocumentData gives a variant of the actor's catalog its SKU, its base
// unit with its DIAN code and its IVA treatment.
func (e *env) setDocumentData(variantID uuid.UUID, sku, baseUnit, baseDIAN string, t vat.Treatment) {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	key := catalogKey{e.actor.TenantID, variantID}
	v := e.store.variants[key]
	v.SKU, v.BaseUnitCode, v.BaseUnitDIANCode, v.TaxTreatment = sku, baseUnit, baseDIAN, t
	e.store.variants[key] = v
}

// setUnitDIANCode gives an alternate unit of a variant its DIAN code.
func (e *env) setUnitDIANCode(variantID, unitID uuid.UUID, code string) {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	key := catalogKey{e.actor.TenantID, variantID}
	v := e.store.variants[key]
	for i := range v.Units {
		if v.Units[i].UnitID == unitID {
			v.Units[i].DIANCode = code
		}
	}
	e.store.variants[key] = v
}

func TestConfirmSaleFreezesDocumentData(t *testing.T) {
	tests := []struct {
		name                 string
		product, price, rate string
		sku, base, baseDIAN  string
		treatment            vat.Treatment
		unit, unitDIAN       string
		wantUnitDIAN         []string
	}{
		{"ferretería: cable por metro y por rollo", "Cable #12", "1800", "0.19", "CAB-12", "m", "MTR", vat.Taxed,
			"rollo", "", []string{"MTR", ""}},
		{"papelería: cuaderno exento por unidad y por paquete", "Cuaderno 100 hojas", "4500", "0", "CUA-100", "und", "94",
			vat.Exempt, "paquete", "PK", []string{"94", "PK"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			variant := e.addProduct(tt.product, tt.price, tt.rate, true)
			e.setDocumentData(variant, tt.sku, tt.base, tt.baseDIAN, tt.treatment)
			unit := e.addUnit(variant, tt.unit, "10", tt.price+"0")
			e.setUnitDIANCode(variant, unit, tt.unitDIAN)

			sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
				LocationID: newID(),
				Lines:      []app.LineRequest{lineReq(variant, "1", "0"), unitLine(variant, unit, "1")},
				Payments:   []app.PaymentRequest{cash("1000000")},
			})
			require.NoError(t, err)

			for i, l := range sale.Lines() {
				require.Equal(t, tt.sku, l.SKU())
				require.Equal(t, tt.base, l.BaseUnitCode())
				require.Equal(t, tt.treatment, l.TaxTreatment())
				require.Equal(t, tt.wantUnitDIAN[i], l.UnitDIANCode(), "línea %d", i)
			}
		})
	}
}
