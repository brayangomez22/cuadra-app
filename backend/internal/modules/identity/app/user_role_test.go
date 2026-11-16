package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestUserRole(t *testing.T) {
	t.Run("devuelve el rol actual del usuario de la empresa", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "dueno@papeleria.co", "clave-segura-123", domain.RoleOwner)
		keeper := e.seedUser(t, tenant.ID(), "bodega@papeleria.co", "clave-segura-123", domain.RoleWarehouse)

		role, err := e.svc.UserRole(t.Context(), tenant.ID(), keeper.ID())
		require.NoError(t, err)
		require.Equal(t, domain.RoleWarehouse, role)
		require.Contains(t, e.span(t, "identity.UserRole").Attributes(), attribute.String("user.id", keeper.ID().String()))
	})

	t.Run("no encuentra a un usuario de otra empresa", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "dueno@drogueria.co", "clave-segura-123", domain.RoleOwner)
		_, other := e.seed(t, "dueno@calzado.co", "clave-segura-123", domain.RoleOwner)

		_, err := e.svc.UserRole(t.Context(), tenant.ID(), other.ID())
		require.ErrorIs(t, err, domain.ErrUserNotFound)
		_, err = e.svc.UserRole(t.Context(), tenant.ID(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
