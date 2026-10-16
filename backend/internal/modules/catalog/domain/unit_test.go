package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestNewUnit(t *testing.T) {
	t.Run("crea una unidad con el código en minúsculas y sin espacios alrededor", func(t *testing.T) {
		tests := []struct {
			name, code, unitName string
			wantCode, wantName   string
		}{
			{"ferretería: bulto", "  Bulto ", " Bulto ", "bulto", "Bulto"},
			{"almacén de ropa: par", "PAR", "Par", "par", "Par"},
			{"droguería: tableta", "tableta", "Tableta", "tableta", "Tableta"},
			{"con tilde", "galón", "Galón", "galón", "Galón"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tenantID := uuid.Must(uuid.NewV7())

				u, err := domain.NewUnit(tenantID, tt.code, tt.unitName, testNow)
				require.NoError(t, err)

				require.Equal(t, 7, int(u.ID().Version()))
				require.Equal(t, tenantID, u.TenantID())
				require.Equal(t, tt.wantCode, u.Code())
				require.Equal(t, tt.wantName, u.Name())
				require.Equal(t, testNow.UTC(), u.CreatedAt())
				require.Equal(t, domain.UnitRef{ID: u.ID(), Code: tt.wantCode, Name: tt.wantName}, u.Ref())
			})
		}
	})

	t.Run("rechaza un código vacío, con espacios internos o de más de 16 caracteres", func(t *testing.T) {
		for _, code := range []string{"", "   ", "caja x12", "m\tl", strings.Repeat("a", 17)} {
			_, err := domain.NewUnit(uuid.Must(uuid.NewV7()), code, "Unidad", testNow)
			require.ErrorIs(t, err, domain.ErrInvalidUnitCode, "%q", code)
		}
	})

	t.Run("acepta un código de 16 caracteres", func(t *testing.T) {
		_, err := domain.NewUnit(uuid.Must(uuid.NewV7()), strings.Repeat("á", 16), "Unidad", testNow)
		require.NoError(t, err)
	})

	t.Run("rechaza un nombre vacío o de más de 50 caracteres", func(t *testing.T) {
		for _, name := range []string{"", "  ", strings.Repeat("a", 51)} {
			_, err := domain.NewUnit(uuid.Must(uuid.NewV7()), "und", name, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidUnitName, "%q", name)
		}
	})

	t.Run("rechaza un tenant vacío", func(t *testing.T) {
		_, err := domain.NewUnit(uuid.Nil, "und", "Unidad", testNow)
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
	})
}

func TestUnitUpdate(t *testing.T) {
	t.Run("cambia el código y el nombre", func(t *testing.T) {
		u, err := domain.NewUnit(uuid.Must(uuid.NewV7()), "galon", "Galon", testNow)
		require.NoError(t, err)

		require.NoError(t, u.Update(" Galón ", " Galón "))

		require.Equal(t, "galón", u.Code())
		require.Equal(t, "Galón", u.Name())
	})

	t.Run("si la edición falla, la unidad queda igual", func(t *testing.T) {
		tests := []struct {
			name, code, unitName string
			want                 error
		}{
			{"código inválido", "caja x12", "Caja", domain.ErrInvalidUnitCode},
			{"nombre inválido", "caja", "", domain.ErrInvalidUnitName},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				u, err := domain.NewUnit(uuid.Must(uuid.NewV7()), "und", "Unidad", testNow)
				require.NoError(t, err)

				require.ErrorIs(t, u.Update(tt.code, tt.unitName), tt.want)

				require.Equal(t, "und", u.Code())
				require.Equal(t, "Unidad", u.Name())
			})
		}
	})
}

func TestRehydrateUnit(t *testing.T) {
	t.Run("rechaza datos guardados inválidos", func(t *testing.T) {
		valid := domain.UnitSnapshot{
			ID: uuid.Must(uuid.NewV7()), TenantID: uuid.Must(uuid.NewV7()), Code: "kg", Name: "Kilogramo", CreatedAt: testNow,
		}
		tests := []struct {
			name   string
			mutate func(*domain.UnitSnapshot)
			want   error
		}{
			{"id vacío", func(s *domain.UnitSnapshot) { s.ID = uuid.Nil }, domain.ErrInvalidUnitID},
			{"tenant vacío", func(s *domain.UnitSnapshot) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"código vacío", func(s *domain.UnitSnapshot) { s.Code = "" }, domain.ErrInvalidUnitCode},
			{"nombre vacío", func(s *domain.UnitSnapshot) { s.Name = "" }, domain.ErrInvalidUnitName},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := valid
				tt.mutate(&s)
				_, err := domain.RehydrateUnit(s)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}
