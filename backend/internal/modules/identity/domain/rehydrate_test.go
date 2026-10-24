package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestRehydrateTenant(t *testing.T) {
	createdAt := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)
	valid := func(t *testing.T) domain.TenantSnapshot {
		t.Helper()
		return domain.TenantSnapshot{
			ID:               uuid.Must(uuid.NewV7()),
			Name:             "Ferretería El Tornillo",
			NIT:              mustNIT(t, "900123456-8"),
			Status:           domain.TenantSuspended,
			BusinessTemplate: "ropa_calzado",
			DefaultTaxRate:   decimal.RequireFromString("0.19"),
			CreatedAt:        createdAt,
		}
	}

	t.Run("reconstruye un tenant suspendido", func(t *testing.T) {
		s := valid(t)
		tenant, err := domain.RehydrateTenant(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, tenant.ID())
		require.Equal(t, "Ferretería El Tornillo", tenant.Name())
		require.Equal(t, "900123456-8", tenant.NIT().String())
		require.Equal(t, domain.TenantSuspended, tenant.Status())
		require.False(t, tenant.IsActive())
		require.Equal(t, "ropa_calzado", tenant.BusinessTemplate())
		require.Equal(t, "0.19", tenant.DefaultTaxRate().String())
		require.Equal(t, createdAt, tenant.CreatedAt())
	})

	t.Run("rechaza datos inválidos", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*domain.TenantSnapshot)
			wantErr error
		}{
			{name: "sin id", mutate: func(s *domain.TenantSnapshot) { s.ID = uuid.Nil }, wantErr: domain.ErrInvalidTenantID},
			{name: "sin nombre", mutate: func(s *domain.TenantSnapshot) { s.Name = " " }, wantErr: domain.ErrInvalidTenantName},
			{name: "sin NIT", mutate: func(s *domain.TenantSnapshot) { s.NIT = domain.NIT{} }, wantErr: domain.ErrInvalidNIT},
			{name: "con estado desconocido", mutate: func(s *domain.TenantSnapshot) { s.Status = "deleted" }, wantErr: domain.ErrInvalidTenantStatus},
			{name: "sin plantilla", mutate: func(s *domain.TenantSnapshot) { s.BusinessTemplate = "" }, wantErr: domain.ErrUnknownBusinessTemplate},
			{name: "con IVA por defecto de 100 %", mutate: func(s *domain.TenantSnapshot) { s.DefaultTaxRate = decimal.NewFromInt(1) }, wantErr: domain.ErrInvalidDefaultTaxRate},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := valid(t)
				tt.mutate(&s)
				_, err := domain.RehydrateTenant(s)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})
}

func TestRehydrateUser(t *testing.T) {
	createdAt := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)
	valid := func(t *testing.T) domain.UserSnapshot {
		t.Helper()
		return domain.UserSnapshot{
			ID:           uuid.Must(uuid.NewV7()),
			TenantID:     uuid.Must(uuid.NewV7()),
			Email:        mustEmail(t, "ana@ferreteria.co"),
			Name:         "Ana Gómez",
			PasswordHash: fakeHash,
			Role:         domain.RoleWarehouse,
			Active:       false,
			CreatedAt:    createdAt,
		}
	}

	t.Run("reconstruye un usuario con todos sus campos", func(t *testing.T) {
		s := valid(t)
		user, err := domain.RehydrateUser(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, user.ID())
		require.Equal(t, s.TenantID, user.TenantID())
		require.Equal(t, "ana@ferreteria.co", user.Email().String())
		require.Equal(t, "Ana Gómez", user.Name())
		require.Equal(t, fakeHash, user.PasswordHash())
		require.Equal(t, domain.RoleWarehouse, user.Role())
		require.False(t, user.IsActive())
		require.Equal(t, createdAt, user.CreatedAt())
	})

	t.Run("rechaza datos inválidos", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*domain.UserSnapshot)
			wantErr error
		}{
			{name: "sin id", mutate: func(s *domain.UserSnapshot) { s.ID = uuid.Nil }, wantErr: domain.ErrInvalidUserID},
			{name: "sin tenant", mutate: func(s *domain.UserSnapshot) { s.TenantID = uuid.Nil }, wantErr: domain.ErrInvalidTenantID},
			{name: "sin email", mutate: func(s *domain.UserSnapshot) { s.Email = domain.Email{} }, wantErr: domain.ErrInvalidEmail},
			{name: "sin nombre", mutate: func(s *domain.UserSnapshot) { s.Name = "" }, wantErr: domain.ErrInvalidUserName},
			{name: "sin hash de contraseña", mutate: func(s *domain.UserSnapshot) { s.PasswordHash = "" }, wantErr: domain.ErrEmptyPasswordHash},
			{name: "con rol inválido", mutate: func(s *domain.UserSnapshot) { s.Role = "root" }, wantErr: domain.ErrInvalidRole},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := valid(t)
				tt.mutate(&s)
				_, err := domain.RehydrateUser(s)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})
}
