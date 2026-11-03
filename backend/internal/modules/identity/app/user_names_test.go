package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestUserNames(t *testing.T) {
	t.Run("devuelve el nombre de los usuarios de la empresa entre los pedidos", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "dueno@ferreteria.co", "clave-segura-123", domain.RoleOwner)
		cashier := e.seedUser(t, tenant.ID(), "cajera@ferreteria.co", "clave-segura-123", domain.RoleCashier)
		_, other := e.seed(t, "dueno@almacen.co", "clave-segura-123", domain.RoleOwner)

		names, err := e.svc.UserNames(t.Context(), tenant.ID(), []uuid.UUID{cashier.ID(), owner.ID(), other.ID(), uuid.Must(uuid.NewV7())})
		require.NoError(t, err)

		require.Equal(t, map[uuid.UUID]string{cashier.ID(): cashier.Name(), owner.ID(): owner.Name()}, names)
		require.Contains(t, e.span(t, "identity.UserNames").Attributes(), attribute.String("tenant.id", tenant.ID().String()))
	})

	t.Run("sin ids no pregunta nada", func(t *testing.T) {
		e := newEnv(t)
		names, err := e.svc.UserNames(t.Context(), uuid.Must(uuid.NewV7()), nil)
		require.NoError(t, err)
		require.Empty(t, names)
	})
}
