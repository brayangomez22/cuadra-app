package httpadapter_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
)

func TestProductTaxTreatment(t *testing.T) {
	t.Run("pasa el tratamiento del IVA al caso de uso (cuaderno exento de papelería)", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}
		body := strings.Replace(productBody, `"tax_rate":"0.19"`, `"tax_rate":"0","tax_treatment":"exempt"`, 1)

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, vat.Exempt, uc.input.TaxRate.Treatment())
	})

	t.Run("sin tratamiento lo deduce de la tasa y lo devuelve (cable gravado de ferretería)", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}

		body := decodeMap(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody))

		require.Equal(t, vat.Taxed, uc.input.TaxRate.Treatment())
		require.Equal(t, "taxed", body["tax_treatment"])
	})

	t.Run("un tratamiento que no corresponde a la tasa responde 422 invalid_tax_treatment", func(t *testing.T) {
		uc := &stubUseCases{}
		body := strings.Replace(productBody, `"tax_rate":"0.19"`, `"tax_rate":"0.19","tax_treatment":"excluded"`, 1)

		requireError(t, serve(t, uc, owner, http.MethodPost, "/api/v1/products", body),
			http.StatusUnprocessableEntity, "invalid_tax_treatment")
		require.Empty(t, uc.input.SKU)
	})
}

func TestUnitDIANCode(t *testing.T) {
	t.Run("pasa el código DIAN al caso de uso y lo devuelve", func(t *testing.T) {
		u := mustUnit("par", "Par")
		require.NoError(t, u.SetDIANCode("PR"))
		uc := &stubUseCases{unit: u}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/units", `{"code":"par","name":"Par","dian_code":"pr"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "pr", uc.dianCode)
		require.Equal(t, "PR", decodeMap(t, rec)["dian_code"])
	})

	t.Run("una unidad sin código DIAN no lo trae en la respuesta", func(t *testing.T) {
		uc := &stubUseCases{unit: mustUnit("caja", "Caja")}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/units", `{"code":"caja","name":"Caja"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NotContains(t, decodeMap(t, rec), "dian_code")
	})
}
