package flags_test

import (
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
)

func newClient(t *testing.T, store *fakeStore) *openfeature.Client {
	t.Helper()
	client, err := flags.NewClient(t.Context(), flags.NewProvider(newEvaluator(t, store, newClock())))
	require.NoError(t, err)
	return client
}

func TestProvider(t *testing.T) {
	t.Run("responde por OpenFeature con la excepción del tenant", func(t *testing.T) {
		store := newFakeStore()
		pilot := store.addTenant("Ferretería La Tuerca")
		other := store.addTenant("Calzado Andino")
		store.setException(pilot, "new_pos", true)
		client := newClient(t, store)

		on, err := client.BooleanValue(t.Context(), "new_pos", false, flags.TenantContext(pilot))
		require.NoError(t, err)
		require.True(t, on)

		on, err = client.BooleanValue(t.Context(), "new_pos", false, flags.TenantContext(other))
		require.NoError(t, err)
		require.False(t, on)
	})

	t.Run("una flag desconocida devuelve el valor por defecto seguro", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Droguería Central")
		store.setGlobal("retired", true)
		client := newClient(t, store)

		details, err := client.BooleanValueDetails(t.Context(), "retired", false, flags.TenantContext(tenant))
		require.Error(t, err)
		require.False(t, details.Value)
		require.Equal(t, openfeature.FlagNotFoundCode, details.ErrorCode)
	})

	t.Run("sin tenant en el contexto devuelve el valor por defecto", func(t *testing.T) {
		store := newFakeStore()
		store.setGlobal("new_pos", true)
		client := newClient(t, store)

		details, err := client.BooleanValueDetails(t.Context(), "new_pos", false, openfeature.NewTargetlessEvaluationContext(nil))
		require.Error(t, err)
		require.False(t, details.Value)
		require.Equal(t, openfeature.TargetingKeyMissingCode, details.ErrorCode)
	})

	t.Run("si el store falla devuelve el valor por defecto con el error", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Papelería El Lápiz")
		store.setGlobal("new_pos", true)
		store.fail = errDown
		client := newClient(t, store)

		details, err := client.BooleanValueDetails(t.Context(), "new_pos", false, flags.TenantContext(tenant))
		require.Error(t, err)
		require.False(t, details.Value)
		require.Equal(t, openfeature.ErrorReason, details.Reason)
	})

	t.Run("una flag que no es booleana no existe", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Calzado Andino")
		client := newClient(t, store)

		got, err := client.StringValue(t.Context(), "new_pos", "off", flags.TenantContext(tenant))
		require.Error(t, err)
		require.Equal(t, "off", got)
	})
}
