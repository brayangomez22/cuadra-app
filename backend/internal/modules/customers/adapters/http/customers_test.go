package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestCreateCustomer(t *testing.T) {
	t.Run("traduce el body y responde el cliente creado", func(t *testing.T) {
		c := aCompany(t)
		uc := &stubUseCases{customer: c}
		body := `{"person_type":"legal","document_type":"nit","document_number":"900.123.456-8",
			"name":"Confecciones La Moda S.A.S.","trade_name":"La Moda","email":"compras@lamoda.co",
			"segment_id":"` + c.SegmentID().String() + `","tax_responsibilities":["O-13"]}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/customers", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, domain.CustomerParams{
			PersonType: domain.PersonLegal, DocumentType: domain.DocumentNIT, DocumentNumber: "900.123.456-8",
			Name: "Confecciones La Moda S.A.S.", TradeName: "La Moda", Contact: domain.Contact{Email: "compras@lamoda.co"},
			SegmentID: c.SegmentID(), TaxResponsibilities: []string{"O-13"}, Active: true,
		}, uc.params)
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "900123456-8", got["document_number"])
		require.Equal(t, "Medellín", got["city"])
		require.Equal(t, []any{"O-13"}, got["tax_responsibilities"])
		require.Equal(t, true, got["active"])
	})

	t.Run("no envía los campos opcionales vacíos", func(t *testing.T) {
		uc := &stubUseCases{customer: aPerson(t)}
		body := `{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"Carlos Pérez"}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/customers", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NotContains(t, rec.Body.String(), "trade_name")
		require.NotContains(t, rec.Body.String(), "segment_id")
		require.NotContains(t, rec.Body.String(), "email")
	})

	t.Run("mapea las reglas del dominio a su status y código", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrInvalidDocument, http.StatusUnprocessableEntity, "invalid_document"},
			{domain.ErrInvalidNITCheckDigit, http.StatusUnprocessableEntity, "invalid_nit_check_digit"},
			{domain.ErrLegalPersonRequiresNIT, http.StatusUnprocessableEntity, "legal_person_requires_nit"},
			{domain.ErrInvalidCustomerName, http.StatusUnprocessableEntity, "invalid_customer_name"},
			{domain.ErrInvalidContact, http.StatusUnprocessableEntity, "invalid_customer_contact"},
			{domain.ErrInvalidTaxResponsibilities, http.StatusUnprocessableEntity, "invalid_tax_responsibilities"},
			{domain.ErrSegmentNotFound, http.StatusUnprocessableEntity, "customer_segment_not_found"},
		}
		body := `{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"Carlos Pérez"}`
		for _, tt := range tests {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPost, "/api/v1/customers", body)
			requireError(t, rec, tt.status, tt.code)
		}
	})

	t.Run("un documento repetido responde 409 y dice cuál", func(t *testing.T) {
		doc, err := domain.ParseDocument(domain.DocumentNIT, "900123456-8")
		require.NoError(t, err)
		uc := &stubUseCases{err: &domain.DocumentTakenError{Document: doc}}
		body := `{"person_type":"legal","document_type":"nit","document_number":"900123456-8","name":"La Moda"}`

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/customers", body)

		msg := requireError(t, rec, http.StatusConflict, "customer_document_taken")
		require.Contains(t, msg, "NIT 900123456-8")
	})

	t.Run("sin principal responde 401", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, nil, http.MethodPost, "/api/v1/customers", `{}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestUpdateCustomer(t *testing.T) {
	t.Run("pasa active y responde el cliente", func(t *testing.T) {
		c := aPerson(t)
		uc := &stubUseCases{customer: c}
		body := `{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"Carlos Pérez","active":false}`

		rec := serve(t, uc, &owner, http.MethodPut, "/api/v1/customers/"+c.ID().String(), body)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, c.ID(), uc.id)
		require.False(t, uc.params.Active)
	})

	t.Run("un cliente inexistente responde 404", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCustomerNotFound}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/customers/"+newID().String(), "")
		requireError(t, rec, http.StatusNotFound, "customer_not_found")
	})
}

func TestListCustomers(t *testing.T) {
	t.Run("pasa el texto, el segmento, el estado y la página", func(t *testing.T) {
		segment := newID()
		uc := &stubUseCases{page: app.CustomerPage{Items: []*domain.Customer{aPerson(t)}, Total: 7}}

		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/customers?q=carlos&segment_id="+segment.String()+"&status=all&limit=5&offset=5", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.CustomerQuery{
			Text: "carlos", SegmentID: &segment, Status: domain.StatusAll, Limit: 5, Offset: 5,
		}, uc.query)
		require.Contains(t, rec.Body.String(), `"total":7`)
	})

	t.Run("sin parámetros pide la primera página de los activos", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/customers", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.CustomerQuery{Status: domain.StatusActive, Limit: 20}, uc.query)
		require.Contains(t, rec.Body.String(), `"items":[]`)
	})
}
