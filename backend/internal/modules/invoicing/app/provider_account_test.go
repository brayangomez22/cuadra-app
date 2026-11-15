package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// orgToken is the token of the tenant's organization at the provider.
const orgToken = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"

func ptr[T any](v T) *T { return &v }

func TestProviderAccount(t *testing.T) {
	t.Run("una empresa que no lo ha configurado está en pruebas y sin token", func(t *testing.T) {
		e := newEnv(t)
		got, err := e.svc.ProviderAccount(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, domain.ModeTest, got.Mode)
		require.False(t, got.HasToken())
	})

	t.Run("guarda el modo, el set de pruebas y el token", func(t *testing.T) {
		e := newEnv(t)
		got, err := e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{
			Mode: "habilitation", TestSetID: "set-123", Token: ptr(orgToken),
		})
		require.NoError(t, err)
		require.Equal(t, domain.ModeHabilitation, got.Mode)

		stored := e.store.accounts[e.actor.TenantID]
		require.Equal(t, "set-123", stored.TestSetID)
		require.Equal(t, orgToken, stored.Token.Reveal())
	})

	t.Run("sin token nuevo conserva el que tenía", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{Mode: "test", Token: ptr(orgToken)})
		require.NoError(t, err)
		got, err := e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{Mode: "production"})
		require.NoError(t, err)
		require.Equal(t, domain.ModeProduction, got.Mode)
		require.Equal(t, orgToken, e.store.accounts[e.actor.TenantID].Token.Reveal())
	})

	t.Run("un dato inválido no cambia nada", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{Mode: "habilitation", Token: ptr(orgToken)})
		require.ErrorIs(t, err, domain.ErrMissingTestSet)
		_, err = e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{Mode: "test", Token: ptr("con espacios")})
		require.ErrorIs(t, err, domain.ErrInvalidProviderToken)
		require.NotContains(t, e.store.accounts, e.actor.TenantID)
	})

	t.Run("el token nunca aparece en los spans ni en los logs", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.UpdateProviderAccount(t.Context(), e.actor, app.ProviderAccountInput{Mode: "test", Token: ptr(orgToken)})
		require.NoError(t, err)
		_, err = e.svc.ProviderAccount(t.Context(), e.actor)
		require.NoError(t, err)

		for _, s := range e.spans.Ended() {
			for _, kv := range s.Attributes() {
				require.NotContains(t, kv.Value.String(), orgToken, s.Name())
			}
		}
		require.Equal(t, e.actor.TenantID.String(), e.span(t, "invoicing.UpdateProviderAccount")["tenant.id"].AsString())
		require.Contains(t, e.logs.String(), "provider account updated")
		require.NotContains(t, e.logs.String(), orgToken)
	})
}
