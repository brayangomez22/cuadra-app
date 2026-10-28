package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestListCustomers(t *testing.T) {
	t.Run("por defecto lista solo los activos, por nombre", func(t *testing.T) {
		e := newEnv(t)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		e.addCustomer(t, company("900123456-8", "Almacén Central"))
		inactive := person("79845123", "Beatriz Ruiz")
		inactive.Active = false
		e.addCustomer(t, inactive)

		page, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Limit: 20})
		require.NoError(t, err)

		require.Equal(t, 2, page.Total)
		require.Equal(t, "Almacén Central", page.Items[0].Name())
		require.Equal(t, "Carlos Pérez", page.Items[1].Name())
		_, attrs := e.span(t, "customers.ListCustomers")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("filtra por texto, segmento y estado", func(t *testing.T) {
		e := newEnv(t)
		wholesale := e.addSegment(t, e.actor.TenantID, "Mayorista")
		id := wholesale.ID()
		inSegment := company("900123456-8", "Almacén Central")
		inSegment.SegmentID = &id
		e.addCustomer(t, inSegment)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))

		bySegment, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{SegmentID: &id, Limit: 20})
		require.NoError(t, err)
		require.Equal(t, 1, bySegment.Total)

		byText, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Text: "  carlos ", Limit: 20})
		require.NoError(t, err)
		require.Equal(t, 1, byText.Total)
		require.Equal(t, "Carlos Pérez", byText.Items[0].Name())

		all, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Status: domain.StatusAll, Limit: 1})
		require.NoError(t, err)
		require.Equal(t, 2, all.Total)
		require.Len(t, all.Items, 1)
	})

	t.Run("rechaza un estado desconocido", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Status: "deleted", Limit: 20})
		require.ErrorIs(t, err, app.ErrInvalidStatus)
	})
}

func TestCheckCustomer(t *testing.T) {
	t.Run("acepta un cliente activo del tenant para vender", func(t *testing.T) {
		e := newEnv(t)
		c := e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		require.NoError(t, e.svc.CheckCustomer(t.Context(), e.actor, c.ID()))
	})

	t.Run("rechaza un cliente inactivo o de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		p := person("1020304050", "Carlos Pérez")
		p.Active = false
		inactive := e.addCustomer(t, p)
		require.ErrorIs(t, e.svc.CheckCustomer(t.Context(), e.actor, inactive.ID()), domain.ErrCustomerInactive)

		other := e.actor
		other.TenantID = newID()
		require.ErrorIs(t, e.svc.CheckCustomer(t.Context(), other, inactive.ID()), domain.ErrCustomerNotFound)
	})
}
