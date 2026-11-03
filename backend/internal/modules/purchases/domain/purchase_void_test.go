package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func newPurchase(t *testing.T, payment domain.Payment, lines ...domain.LineInput) *domain.Purchase {
	t.Helper()
	p, err := domain.NewPurchase(tenantID, purchaseParams(t, payment), lines, now)
	require.NoError(t, err)
	return p
}

func TestPurchaseVoid(t *testing.T) {
	later := now.Add(2 * time.Hour)

	t.Run("anula una compra a crédito sin pagos: no debe nada y guarda quién, cuándo y por qué", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))

		require.NoError(t, p.Void(userID, "  Factura duplicada  ", later))

		require.Equal(t, domain.PurchaseVoided, p.Status())
		require.True(t, p.BalanceDue().IsZero())
		v := p.Voided()
		require.NotNil(t, v)
		require.Equal(t, userID, v.By)
		require.Equal(t, "Factura duplicada", v.Reason)
		require.Equal(t, later, v.At)
	})

	t.Run("anula una compra de contado de un almacén de ropa", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCash, sockDozens("3", "54000"))
		require.NoError(t, p.Void(userID, "Llegó otra talla", later))
		require.Equal(t, domain.PurchaseVoided, p.Status())
	})

	t.Run("no anula dos veces", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCash, screwBoxes("1", "45000"))
		require.NoError(t, p.Void(userID, "Error", later))
		require.ErrorIs(t, p.Void(userID, "Otra vez", later), domain.ErrPurchaseAlreadyVoided)
	})

	t.Run("no anula una compra a crédito que ya tiene pagos", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
		_, err := p.Pay(aPayment("10000"), later)
		require.NoError(t, err)

		require.ErrorIs(t, p.Void(userID, "Error", later), domain.ErrPurchaseHasPayments)
		require.Equal(t, domain.PurchaseOpen, p.Status())
	})

	t.Run("exige un motivo de máximo 500 caracteres y quién anula", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCash, screwBoxes("1", "45000"))
		require.ErrorIs(t, p.Void(userID, "   ", later), domain.ErrInvalidVoidReason)
		require.ErrorIs(t, p.Void(userID, strings.Repeat("a", 501), later), domain.ErrInvalidVoidReason)
		require.ErrorIs(t, p.Void(uuid.Nil, "Error", later), domain.ErrInvalidUserID)
		require.Nil(t, p.Voided())
	})

	t.Run("una compra anulada se guarda y se reconstruye anulada", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
		require.NoError(t, p.Receive([]domain.Received{{BaseQuantity: dec("200"), BaseUnitCost: dec("450")}}))
		require.NoError(t, p.Void(userID, "Error", later))

		back, err := domain.RehydratePurchase(p.Snapshot())
		require.NoError(t, err)
		require.Equal(t, domain.PurchaseVoided, back.Status())
		require.Equal(t, p.Voided(), back.Voided())

		bad := p.Snapshot()
		bad.BalanceDue = dec("100")
		_, err = domain.RehydratePurchase(bad)
		require.ErrorIs(t, err, domain.ErrInvalidPurchase)
	})
}
