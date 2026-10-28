package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestCreateCustomer(t *testing.T) {
	t.Run("crea clientes en segmentos de distintos tipos de negocio", func(t *testing.T) {
		tests := []struct {
			name, segment string
			params        domain.CustomerParams
		}{
			{"ferretería: maestro de obra", "Maestro de obra", person("1020304050", "Carlos Pérez")},
			{"almacén de ropa: mayorista", "Mayorista", company("900123456-8", "Confecciones La Moda S.A.S.")},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				segment := e.addSegment(t, e.actor.TenantID, tt.segment)
				p := tt.params
				id := segment.ID()
				p.SegmentID = &id

				c, err := e.svc.CreateCustomer(t.Context(), e.actor, p)
				require.NoError(t, err)

				require.Equal(t, segment.ID(), *c.SegmentID())
				stored, err := e.svc.GetCustomer(t.Context(), e.actor, c.ID())
				require.NoError(t, err)
				require.Equal(t, c, stored)
			})
		}
	})

	t.Run("abre su span y cuenta el cliente por tipo de documento", func(t *testing.T) {
		e := newEnv(t)
		c := e.addCustomer(t, company("900123456-8", "Confecciones La Moda S.A.S."))

		_, attrs := e.span(t, "customers.CreateCustomer")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, c.ID().String(), attrs["customer.id"].AsString())
		require.EqualValues(t, 1, e.counter(t, "cuadra.customers.created", "document_type", "nit"))
	})

	t.Run("no permite dos clientes con el mismo documento en el tenant", func(t *testing.T) {
		e := newEnv(t)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))

		_, err := e.svc.CreateCustomer(t.Context(), e.actor, person("1.020.304.050", "Otro nombre"))

		var taken *domain.DocumentTakenError
		require.ErrorAs(t, err, &taken)
		require.ErrorIs(t, err, domain.ErrDocumentTaken)
		require.Equal(t, "CC 1020304050", taken.Document.String())
		span, _ := e.span(t, "customers.CreateCustomer")
		require.Equal(t, codes.Error, span.Status().Code)
	})

	t.Run("el mismo documento sí puede existir en otro tenant", func(t *testing.T) {
		e := newEnv(t)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		other := e.actor
		other.TenantID = newID()

		_, err := e.svc.CreateCustomer(t.Context(), other, person("1020304050", "Carlos Pérez"))
		require.NoError(t, err)
	})

	t.Run("rechaza un segmento que no es del tenant", func(t *testing.T) {
		e := newEnv(t)
		foreign := e.addSegment(t, newID(), "Mayorista")
		p := person("1020304050", "Carlos Pérez")
		id := foreign.ID()
		p.SegmentID = &id

		_, err := e.svc.CreateCustomer(t.Context(), e.actor, p)
		require.ErrorIs(t, err, domain.ErrSegmentNotFound)
	})

	t.Run("rechaza NIT con DV inválido", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.CreateCustomer(t.Context(), e.actor, company("900123456-1", "Confecciones La Moda S.A.S."))
		require.ErrorIs(t, err, domain.ErrInvalidNITCheckDigit)
		require.EqualValues(t, 0, e.counter(t, "cuadra.customers.created", "document_type", "nit"))
	})
}

func TestUpdateCustomer(t *testing.T) {
	t.Run("edita y desactiva un cliente", func(t *testing.T) {
		e := newEnv(t)
		c := e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		p := person("1020304050", "Carlos Andrés Pérez")
		p.Active = false

		updated, err := e.svc.UpdateCustomer(t.Context(), e.actor, c.ID(), p)
		require.NoError(t, err)

		require.Equal(t, "Carlos Andrés Pérez", updated.Name())
		require.False(t, updated.Active())
		require.Equal(t, c.CreatedAt(), updated.CreatedAt())
		_, attrs := e.span(t, "customers.UpdateCustomer")
		require.Equal(t, c.ID().String(), attrs["customer.id"].AsString())
	})

	t.Run("no permite tomar el documento de otro cliente", func(t *testing.T) {
		e := newEnv(t)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		c := e.addCustomer(t, person("79845123", "Ana Gómez"))

		_, err := e.svc.UpdateCustomer(t.Context(), e.actor, c.ID(), person("1020304050", "Ana Gómez"))
		require.ErrorIs(t, err, domain.ErrDocumentTaken)
	})

	t.Run("no encuentra el cliente de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		c := e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		other := e.actor
		other.TenantID = newID()

		_, err := e.svc.UpdateCustomer(t.Context(), other, c.ID(), person("1020304050", "Carlos Pérez"))
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
		_, err = e.svc.GetCustomer(t.Context(), other, c.ID())
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}
