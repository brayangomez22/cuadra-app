package modules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// testDefinitions are modules of different kinds of business: variants
// (clothing and footwear), lots that depend on variants (pharmacies) and
// work orders (repair shops).
func testDefinitions() []modules.Definition {
	return []modules.Definition{
		{ID: "variants", Name: "Variantes"},
		{
			ID: "lots_expiry", Name: "Lotes y vencimientos", DependsOn: []string{"variants"},
			Permissions: []modules.Permission{{ID: "lots:manage", Roles: []string{"admin", "warehouse"}}},
		},
		{
			ID: "work_orders", Name: "Órdenes de trabajo",
			Permissions: []modules.Permission{{ID: "work_orders:write", Roles: []string{"cashier"}}},
		},
	}
}

func TestNewRegistry(t *testing.T) {
	t.Run("registra los módulos en el orden declarado", func(t *testing.T) {
		r, err := modules.NewRegistry(testDefinitions()...)
		require.NoError(t, err)

		var ids []string
		for _, d := range r.Definitions() {
			ids = append(ids, d.ID)
		}
		require.Equal(t, []string{"variants", "lots_expiry", "work_orders"}, ids)
		require.True(t, r.Known("lots_expiry"))
		require.False(t, r.Known("price_lists"))
	})

	tests := []struct {
		name    string
		defs    []modules.Definition
		wantErr error
	}{
		{
			name:    "rechaza un registro con dos módulos con el mismo ID",
			defs:    []modules.Definition{{ID: "variants", Name: "Variantes"}, {ID: "variants", Name: "Otra"}},
			wantErr: modules.ErrDuplicateModule,
		},
		{
			name:    "rechaza una dependencia que no está registrada",
			defs:    []modules.Definition{{ID: "lots_expiry", Name: "Lotes", DependsOn: []string{"variants"}}},
			wantErr: modules.ErrUnknownDependency,
		},
		{
			name:    "rechaza que un módulo dependa de sí mismo",
			defs:    []modules.Definition{{ID: "variants", Name: "Variantes", DependsOn: []string{"variants"}}},
			wantErr: modules.ErrDependencyCycle,
		},
		{
			name: "rechaza dependencias circulares",
			defs: []modules.Definition{
				{ID: "quotes", Name: "Cotizaciones", DependsOn: []string{"price_lists"}},
				{ID: "price_lists", Name: "Listas de precios", DependsOn: []string{"volume_pricing"}},
				{ID: "volume_pricing", Name: "Precios por cantidad", DependsOn: []string{"quotes"}},
			},
			wantErr: modules.ErrDependencyCycle,
		},
		{
			name:    "rechaza un ID con mayúsculas o espacios",
			defs:    []modules.Definition{{ID: "Lots expiry", Name: "Lotes"}},
			wantErr: modules.ErrInvalidModuleID,
		},
		{
			name:    "rechaza un módulo sin nombre",
			defs:    []modules.Definition{{ID: "variants"}},
			wantErr: modules.ErrInvalidModuleName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := modules.NewRegistry(tt.defs...)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("modificar las definiciones después de registrarlas no altera el registro", func(t *testing.T) {
		defs := testDefinitions()
		r, err := modules.NewRegistry(defs...)
		require.NoError(t, err)
		defs[1].DependsOn[0] = "work_orders"
		defs[1].Permissions[0].Roles[0] = "cashier"

		got := r.Definitions()[1]
		require.Equal(t, []string{"variants"}, got.DependsOn)
		require.Equal(t, []string{"admin", "warehouse"}, got.Permissions[0].Roles)
	})
}

func TestBuiltin(t *testing.T) {
	t.Run("los módulos de Cuadra forman un registro válido con variants", func(t *testing.T) {
		r, err := modules.NewRegistry(modules.Builtin()...)
		require.NoError(t, err)
		require.True(t, r.Known(modules.Variants))
	})
}
