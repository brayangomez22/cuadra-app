package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestBuyerDIANData(t *testing.T) {
	t.Run("el consumidor final va con el código DIAN de la cédula", func(t *testing.T) {
		b := domain.FinalConsumer()
		require.Equal(t, "13", b.IdentificationCode)
		require.Equal(t, domain.FinalConsumerDocument, b.DocumentNumber)
	})

	t.Run("el departamento sale del municipio DANE (Medellín y Bogotá)", func(t *testing.T) {
		require.Equal(t, "05", domain.Buyer{MunicipalityCode: "05001"}.DepartmentCode())
		require.Equal(t, "11", domain.Buyer{MunicipalityCode: "11001"}.DepartmentCode())
	})

	t.Run("un comprador con ciudad en texto libre no tiene departamento", func(t *testing.T) {
		require.Empty(t, domain.Buyer{City: "Pereira"}.DepartmentCode())
	})
}
