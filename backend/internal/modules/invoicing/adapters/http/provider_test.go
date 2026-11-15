package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

const orgToken = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"

func withToken(t *testing.T, mode, testSet string) domain.ProviderAccount {
	t.Helper()
	s, err := domain.NewSecret(orgToken)
	require.NoError(t, err)
	a, err := domain.NewProviderAccount(mode, testSet, s)
	require.NoError(t, err)
	return a
}

func TestProviderAccount(t *testing.T) {
	t.Run("responde el modo y si tiene token, nunca el token", func(t *testing.T) {
		uc := stub{account: func(a app.Actor) (domain.ProviderAccount, error) {
			require.Equal(t, owner.TenantID, a.TenantID)
			return withToken(t, "habilitation", "set-123"), nil
		}}

		rec := serve(t, uc, http.MethodGet, "/api/v1/invoicing/provider", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"mode":"habilitation","test_set_id":"set-123","has_token":true}`, rec.Body.String())
	})

	t.Run("pasa el token al caso de uso y no lo devuelve", func(t *testing.T) {
		var got app.ProviderAccountInput
		uc := stub{setAccount: func(_ app.Actor, in app.ProviderAccountInput) (domain.ProviderAccount, error) {
			got = in
			return withToken(t, "production", ""), nil
		}}

		rec := serve(t, uc, http.MethodPut, "/api/v1/invoicing/provider", `{"mode":"production","token":"`+orgToken+`"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "production", got.Mode)
		require.Equal(t, orgToken, *got.Token)
		require.NotContains(t, rec.Body.String(), orgToken)
		require.Contains(t, rec.Body.String(), `"has_token":true`)
	})

	t.Run("sin token conserva el que había", func(t *testing.T) {
		var got app.ProviderAccountInput
		uc := stub{setAccount: func(_ app.Actor, in app.ProviderAccountInput) (domain.ProviderAccount, error) {
			got = in
			return domain.DefaultProviderAccount(), nil
		}}

		rec := serve(t, uc, http.MethodPut, "/api/v1/invoicing/provider", `{"mode":"test"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Nil(t, got.Token)
	})

	t.Run("los errores del dominio responden 422 con su código", func(t *testing.T) {
		cases := []struct {
			err  error
			code string
		}{
			{domain.ErrMissingTestSet, "missing_test_set"},
			{domain.ErrInvalidTestSet, "invalid_test_set"},
			{domain.ErrInvalidProviderToken, "invalid_provider_token"},
			{domain.ErrInvalidProviderMode, "invalid_provider_mode"},
		}
		for _, tc := range cases {
			uc := stub{setAccount: func(app.Actor, app.ProviderAccountInput) (domain.ProviderAccount, error) {
				return domain.ProviderAccount{}, tc.err
			}}
			rec := serve(t, uc, http.MethodPut, "/api/v1/invoicing/provider", `{"mode":"habilitation"}`)
			requireError(t, rec, http.StatusUnprocessableEntity, tc.code)
		}
	})
}
