package flags_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
)

const ttl = 30 * time.Second

func newEvaluator(t *testing.T, store *fakeStore, c *clock) *flags.Evaluator {
	t.Helper()
	registry, err := flags.NewRegistry(
		flags.Definition{Key: "new_pos", Description: "POS nuevo"},
		flags.Definition{Key: "quick_sale", Description: "Venta rápida"},
	)
	require.NoError(t, err)
	e, err := flags.NewEvaluator(flags.EvaluatorDeps{Registry: registry, Store: store, TTL: ttl, Now: c.Now})
	require.NoError(t, err)
	return e
}

func TestEvaluatorEnabled(t *testing.T) {
	// A hardware store and a clothing store: the flags do not care what the
	// tenant sells, only whether it has an exception.
	cases := []struct {
		name      string
		global    *bool
		exception *bool
		want      bool
	}{
		{"la excepción del tenant prende lo que está apagado para todos", new(false), new(true), true},
		{"la excepción del tenant apaga lo que está prendido para todos", new(true), new(false), false},
		{"sin excepción, el tenant sigue el valor global prendido", new(true), nil, true},
		{"sin excepción, el tenant sigue el valor global apagado", new(false), nil, false},
		{"sin valor global ni excepción, la flag está apagada", nil, nil, false},
		{"la excepción vale aunque no haya valor global", nil, new(true), true},
	}
	for _, business := range []string{"Ferretería La Tuerca", "Calzado Andino"} {
		for _, tc := range cases {
			t.Run(business+": "+tc.name, func(t *testing.T) {
				store := newFakeStore()
				tenant := store.addTenant(business)
				if tc.global != nil {
					store.setGlobal("new_pos", *tc.global)
				}
				if tc.exception != nil {
					store.setException(tenant, "new_pos", *tc.exception)
				}
				e := newEvaluator(t, store, newClock())

				require.Equal(t, tc.want, e.Enabled(t.Context(), tenant, "new_pos"))
			})
		}
	}

	t.Run("una flag desconocida está apagada aunque haya una fila con su llave", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Droguería Central")
		store.setGlobal("retired", true)
		store.setException(tenant, "retired", true)
		e := newEvaluator(t, store, newClock())

		require.False(t, e.Enabled(t.Context(), tenant, "retired"))
	})

	t.Run("si el store falla, la flag queda apagada", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Papelería El Lápiz")
		store.setGlobal("new_pos", true)
		store.fail = errDown
		e := newEvaluator(t, store, newClock())

		require.False(t, e.Enabled(t.Context(), tenant, "new_pos"))
	})
}

func TestEvaluatorCache(t *testing.T) {
	t.Run("un cambio se refleja después de la expiración de la caché", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Ferretería La Tuerca")
		c := newClock()
		e := newEvaluator(t, store, c)
		require.False(t, e.Enabled(t.Context(), tenant, "new_pos"))

		store.setGlobal("new_pos", true)
		c.advance(ttl - time.Second)
		require.False(t, e.Enabled(t.Context(), tenant, "new_pos"), "todavía en caché")

		c.advance(time.Second)
		require.True(t, e.Enabled(t.Context(), tenant, "new_pos"))
	})

	t.Run("dentro de la expiración no vuelve a leer el store", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Calzado Andino")
		e := newEvaluator(t, store, newClock())

		e.Enabled(t.Context(), tenant, "new_pos")
		reads := store.readCount()
		e.Enabled(t.Context(), tenant, "quick_sale")
		e.Enabled(t.Context(), tenant, "new_pos")

		require.Equal(t, reads, store.readCount())
	})

	t.Run("invalidar la caché refleja el cambio de inmediato", func(t *testing.T) {
		store := newFakeStore()
		tenant := store.addTenant("Calzado Andino")
		e := newEvaluator(t, store, newClock())
		require.False(t, e.Enabled(t.Context(), tenant, "new_pos"))

		store.setException(tenant, "new_pos", true)
		e.Invalidate()

		require.True(t, e.Enabled(t.Context(), tenant, "new_pos"))
	})
}

func TestEvaluatorActive(t *testing.T) {
	t.Run("lista las flags prendidas del tenant en el orden en que se declararon", func(t *testing.T) {
		store := newFakeStore()
		pilot := store.addTenant("Ferretería La Tuerca")
		other := store.addTenant("Calzado Andino")
		store.setGlobal("quick_sale", true)
		store.setException(pilot, "new_pos", true)
		store.setGlobal("retired", true)
		e := newEvaluator(t, store, newClock())

		got, err := e.Active(t.Context(), pilot)
		require.NoError(t, err)
		require.Equal(t, []string{"new_pos", "quick_sale"}, got)

		got, err = e.Active(t.Context(), other)
		require.NoError(t, err)
		require.Equal(t, []string{"quick_sale"}, got)
	})

	t.Run("si el store falla, devuelve el error", func(t *testing.T) {
		store := newFakeStore()
		store.fail = errDown
		e := newEvaluator(t, store, newClock())

		_, err := e.Active(t.Context(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, errDown)
	})
}
