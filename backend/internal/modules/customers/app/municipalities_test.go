package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// fakeMunicipalities is a DANE catalog of two municipalities.
type fakeMunicipalities struct{}

func (fakeMunicipalities) List(context.Context) ([]domain.Municipality, error) {
	return []domain.Municipality{
		{Code: "05001", Name: "Medellín", DepartmentCode: "05", DepartmentName: "Antioquia"},
		{Code: "11001", Name: "Bogotá, D.C.", DepartmentCode: "11", DepartmentName: "Bogotá, D.C."},
	}, nil
}

func TestListMunicipalities(t *testing.T) {
	t.Run("lista los municipios del DANE con el actor en el span", func(t *testing.T) {
		e := newEnv(t)

		got, err := e.svc.ListMunicipalities(t.Context(), e.actor)
		require.NoError(t, err)

		require.Len(t, got, 2)
		require.Equal(t, "Medellín", got[0].Name)
		_, attrs := e.span(t, "customers.ListMunicipalities")
		require.Equal(t, attribute.StringValue(e.actor.TenantID.String()), attrs["tenant.id"])
	})
}
