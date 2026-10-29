package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

var (
	locationID = uuid.MustParse("01920000-0000-7000-8000-000000000101")
	userID     = uuid.MustParse("01920000-0000-7000-8000-000000000201")
	screwsID   = uuid.MustParse("01920000-0000-7000-8000-000000000301")
	socksID    = uuid.MustParse("01920000-0000-7000-8000-000000000302")
	boxID      = uuid.MustParse("01920000-0000-7000-8000-000000000401")
	dozenID    = uuid.MustParse("01920000-0000-7000-8000-000000000402")
	today      = domain.NewDate(2026, time.October, 28)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// aSupplier is an active supplier with a 30-day payment term.
func aSupplier(t *testing.T) *domain.Supplier {
	t.Helper()
	s, err := domain.NewSupplier(tenantID, supplierParams("Distribuidora del Valle", "900123456-8"), now)
	require.NoError(t, err)
	return s
}

// screwBoxes is a hardware store's line: boxes of 100 screws.
func screwBoxes(qty, cost string) domain.LineInput {
	return domain.LineInput{
		VariantID: screwsID, ProductName: "Tornillo drywall 6x1", Quantity: dec(qty), UnitCost: dec(cost),
		TaxRate: dec("0.19"), Unit: domain.LineUnit{ID: &boxID, Code: "caja", Factor: dec("100")},
	}
}

// sockDozens is a clothing store's line: dozens of socks, size M.
func sockDozens(qty, cost string) domain.LineInput {
	return domain.LineInput{
		VariantID: socksID, ProductName: "Medias deportivas", VariantLabel: "M / Blanco", Quantity: dec(qty),
		UnitCost: dec(cost), TaxRate: dec("0.19"), Unit: domain.LineUnit{ID: &dozenID, Code: "docena", Factor: dec("12")},
	}
}

func purchaseParams(t *testing.T, payment domain.Payment) domain.PurchaseParams {
	t.Helper()
	return domain.PurchaseParams{
		Supplier: aSupplier(t), LocationID: locationID, InvoiceNumber: "FE 1234", IssuedOn: today.AddDays(-2),
		Payment: payment, CreatedBy: userID,
	}
}
