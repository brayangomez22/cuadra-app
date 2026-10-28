package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestConfirmSaleCustomer(t *testing.T) {
	t.Run("guarda el cliente de la venta", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Camiseta básica", "10000", "0.19", true)
		customer := e.addCustomer(true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(), CustomerID: &customer,
			Lines:    []app.LineRequest{lineReq(id, "1", "0")},
			Payments: []app.PaymentRequest{cash("11900")},
		})
		require.NoError(t, err)
		require.Equal(t, customer, *sale.CustomerID())
	})

	t.Run("una venta sin cliente es a consumidor final", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Cemento gris 50kg", "30000", "0.19", true)

		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(id, "1", "0")},
			Payments:   []app.PaymentRequest{cash("35700")},
		})
		require.NoError(t, err)
		require.Nil(t, sale.CustomerID())
	})

	t.Run("rechaza la venta a un cliente inexistente o inactivo sin guardar ni reservar", func(t *testing.T) {
		tests := []struct {
			name    string
			active  bool
			exists  bool
			wantErr error
		}{
			{name: "inexistente", exists: false, wantErr: domain.ErrCustomerNotFound},
			{name: "inactivo", exists: true, active: false, wantErr: domain.ErrCustomerInactive},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				id := e.addProduct("Camiseta básica", "10000", "0.19", true)
				customer := newID()
				if tt.exists {
					customer = e.addCustomer(tt.active)
				}

				_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
					LocationID: newID(), CustomerID: &customer,
					Lines:    []app.LineRequest{lineReq(id, "1", "0")},
					Payments: []app.PaymentRequest{cash("11900")},
				})
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, e.store.sales)
				require.Empty(t, e.store.reserved)
			})
		}
	})
}
