package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestListMunicipalities(t *testing.T) {
	t.Run("lista los municipios del DANE", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/municipalities", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items":[{"code":"05001","name":"Medellín","department_code":"05","department_name":"Antioquia"}]}`,
			rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
	})
}

func TestCustomerDIANData(t *testing.T) {
	t.Run("pasa el municipio y los tipos de documento nuevos al caso de uso (almacén de ropa con cliente extranjero)", func(t *testing.T) {
		uc := &stubUseCases{customer: aCompany(t)}
		body := `{"person_type":"legal","document_type":"foreign_nit","document_number":"B12345678",
			"name":"Textiles Ibéricos S.L.","municipality_code":"11001"}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/customers", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, domain.DocumentForeignNIT, uc.params.DocumentType)
		require.Equal(t, "11001", uc.params.Contact.MunicipalityCode)
	})

	t.Run("responde el municipio del cliente (ferretería en Medellín)", func(t *testing.T) {
		p := domain.CustomerParams{
			PersonType: domain.PersonNatural, DocumentType: domain.DocumentTI, DocumentNumber: "1098765432",
			Name: "Juan Pérez", Contact: domain.Contact{MunicipalityCode: "05001"}, Active: true,
		}
		c, err := domain.NewCustomer(owner.TenantID, p, now)
		require.NoError(t, err)
		uc := &stubUseCases{customer: c}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/customers/"+c.ID().String(), "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "05001", got["municipality_code"])
		require.Equal(t, "ti", got["document_type"])
	})

	t.Run("un municipio que no existe responde 422 invalid_municipality", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrInvalidMunicipality}
		body := `{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"Ana",
			"municipality_code":"99999"}`

		requireError(t, serve(t, uc, &owner, http.MethodPost, "/api/v1/customers", body),
			http.StatusUnprocessableEntity, "invalid_municipality")
	})
}
