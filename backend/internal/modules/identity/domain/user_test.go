package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

const fakeHash = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA"

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	email, err := domain.ParseEmail(s)
	require.NoError(t, err)
	return email
}

func TestNewUser(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.UTC)
	tenantID := uuid.Must(uuid.NewV7())

	t.Run("crea usuario activo con email normalizado", func(t *testing.T) {
		user, err := domain.NewUser(tenantID, mustEmail(t, "Caja1@Ferreteria.co"), "  Ana Gómez ", fakeHash, domain.RoleCashier, now)
		require.NoError(t, err)
		require.Equal(t, 7, int(user.ID().Version()))
		require.Equal(t, tenantID, user.TenantID())
		require.Equal(t, "caja1@ferreteria.co", user.Email().String())
		require.Equal(t, "Ana Gómez", user.Name())
		require.Equal(t, fakeHash, user.PasswordHash())
		require.Equal(t, domain.RoleCashier, user.Role())
		require.True(t, user.IsActive())
		require.Equal(t, now, user.CreatedAt())
	})

	t.Run("rechaza usuario con datos inválidos", func(t *testing.T) {
		email := mustEmail(t, "ana@ferreteria.co")
		tests := []struct {
			name     string
			tenantID uuid.UUID
			email    domain.Email
			userName string
			hash     string
			role     domain.Role
			wantErr  error
		}{
			{name: "sin tenant", tenantID: uuid.Nil, email: email, userName: "Ana", hash: fakeHash, role: domain.RoleAdmin, wantErr: domain.ErrInvalidTenantID},
			{name: "sin email", tenantID: tenantID, email: domain.Email{}, userName: "Ana", hash: fakeHash, role: domain.RoleAdmin, wantErr: domain.ErrInvalidEmail},
			{name: "sin nombre", tenantID: tenantID, email: email, userName: "  ", hash: fakeHash, role: domain.RoleAdmin, wantErr: domain.ErrInvalidUserName},
			{name: "nombre demasiado largo", tenantID: tenantID, email: email, userName: strings.Repeat("a", 201), hash: fakeHash, role: domain.RoleAdmin, wantErr: domain.ErrInvalidUserName},
			{name: "sin hash de contraseña", tenantID: tenantID, email: email, userName: "Ana", hash: "", role: domain.RoleAdmin, wantErr: domain.ErrEmptyPasswordHash},
			{name: "con rol inválido", tenantID: tenantID, email: email, userName: "Ana", hash: fakeHash, role: domain.Role("root"), wantErr: domain.ErrInvalidRole},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewUser(tt.tenantID, tt.email, tt.userName, tt.hash, tt.role, now)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})

	t.Run("usuario inactivo no tiene permisos", func(t *testing.T) {
		user, err := domain.NewUser(tenantID, mustEmail(t, "dueno@ferreteria.co"), "Dueño", fakeHash, domain.RoleOwner, now)
		require.NoError(t, err)
		require.True(t, user.Can(domain.PermUsersManage))

		user.Deactivate()
		require.False(t, user.IsActive())
		require.False(t, user.Can(domain.PermCatalogRead))

		user.Activate()
		require.True(t, user.Can(domain.PermCatalogRead))
	})

	t.Run("cambia el rol y con él los permisos", func(t *testing.T) {
		user, err := domain.NewUser(tenantID, mustEmail(t, "ana@ferreteria.co"), "Ana", fakeHash, domain.RoleCashier, now)
		require.NoError(t, err)
		require.False(t, user.Can(domain.PermCatalogWrite))

		require.NoError(t, user.ChangeRole(domain.RoleAdmin))
		require.True(t, user.Can(domain.PermCatalogWrite))

		require.ErrorIs(t, user.ChangeRole(domain.Role("root")), domain.ErrInvalidRole)
		require.Equal(t, domain.RoleAdmin, user.Role())
	})
}
