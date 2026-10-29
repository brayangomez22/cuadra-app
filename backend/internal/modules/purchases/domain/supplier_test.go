package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

var (
	tenantID = uuid.MustParse("01920000-0000-7000-8000-000000000001")
	now      = time.Date(2026, time.October, 28, 15, 0, 0, 0, time.UTC)
)

func supplierParams(name, nit string) domain.SupplierParams {
	return domain.SupplierParams{
		Name: name, NIT: nit, PaymentTermDays: 30, Active: true,
		Contact: domain.Contact{Name: "Laura Pérez", Phone: "+57 604 444 5566", Email: "Ventas@Proveedor.co"},
	}
}

func TestNewSupplier(t *testing.T) {
	t.Run("crea proveedores de distintos tipos de negocio con su NIT normalizado", func(t *testing.T) {
		for _, tc := range []struct{ name, nit, want string }{
			{"Distribuidora de Tornillos del Valle", "900.123.456-8", "900123456-8"},
			{"Confecciones Medias Andinas", "9012345677", "901234567-7"},
		} {
			s, err := domain.NewSupplier(tenantID, supplierParams(tc.name, tc.nit), now)
			require.NoError(t, err)
			require.Equal(t, tc.want, s.NIT().String())
			require.Equal(t, tc.name, s.Name())
			require.Equal(t, "ventas@proveedor.co", s.Contact().Email)
			require.Equal(t, 30, s.PaymentTermDays())
		}
	})

	t.Run("rechaza proveedor con NIT de dígito de verificación inválido", func(t *testing.T) {
		_, err := domain.NewSupplier(tenantID, supplierParams("Droguería Mayorista", "900123456-1"), now)
		require.ErrorIs(t, err, domain.ErrInvalidNITCheckDigit)
	})

	t.Run("rechaza proveedor sin NIT o sin nombre", func(t *testing.T) {
		_, err := domain.NewSupplier(tenantID, supplierParams("Papelería al Por Mayor", ""), now)
		require.ErrorIs(t, err, domain.ErrInvalidNIT)
		_, err = domain.NewSupplier(tenantID, supplierParams("  ", "900123456-8"), now)
		require.ErrorIs(t, err, domain.ErrInvalidSupplierName)
	})

	t.Run("rechaza plazo de pago negativo o mayor a 365 días", func(t *testing.T) {
		for _, days := range []int{-1, 366} {
			p := supplierParams("Calzado del Norte", "900123456-8")
			p.PaymentTermDays = days
			_, err := domain.NewSupplier(tenantID, p, now)
			require.ErrorIs(t, err, domain.ErrInvalidPaymentTerm)
		}
	})

	t.Run("rechaza contacto con correo o teléfono mal escrito", func(t *testing.T) {
		p := supplierParams("Calzado del Norte", "900123456-8")
		p.Contact.Email = "ventas-sin-arroba"
		_, err := domain.NewSupplier(tenantID, p, now)
		require.ErrorIs(t, err, domain.ErrInvalidContact)
	})

	t.Run("editar con datos inválidos deja el proveedor como estaba", func(t *testing.T) {
		s, err := domain.NewSupplier(tenantID, supplierParams("Calzado del Norte", "900123456-8"), now)
		require.NoError(t, err)
		bad := supplierParams("", "900123456-8")
		require.ErrorIs(t, s.Update(bad), domain.ErrInvalidSupplierName)
		require.Equal(t, "Calzado del Norte", s.Name())
	})

	t.Run("se reconstruye igual desde lo guardado", func(t *testing.T) {
		s, err := domain.NewSupplier(tenantID, supplierParams("Calzado del Norte", "900123456-8"), now)
		require.NoError(t, err)
		back, err := domain.RehydrateSupplier(s.Snapshot())
		require.NoError(t, err)
		require.Equal(t, s.Snapshot(), back.Snapshot())
	})
}
