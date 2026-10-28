package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleCustomer(t *testing.T) {
	t.Run("pasa el cliente al caso de uso y lo responde en la venta", func(t *testing.T) {
		customer := newID()
		sale, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, CustomerID: &customer, CreatedAt: now,
		})
		require.NoError(t, err)
		uc := &stubUseCases{sale: sale}
		body := `{"location_id":"` + newID().String() + `","customer_id":"` + customer.String() +
			`","lines":[{"variant_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/sales", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, customer, *uc.confirm.CustomerID)
		require.Contains(t, rec.Body.String(), `"customer_id":"`+customer.String()+`"`)
	})

	t.Run("una venta a consumidor final no envía cliente", func(t *testing.T) {
		uc := &stubUseCases{sale: aSale(t)}
		body := `{"location_id":"` + newID().String() + `","lines":[{"variant_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/sales", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Nil(t, uc.confirm.CustomerID)
		require.NotContains(t, rec.Body.String(), "customer_id")
	})
}
