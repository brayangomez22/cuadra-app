package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestProductUnit(t *testing.T) {
	t.Run("crea un producto con la unidad base del tenant por id", func(t *testing.T) {
		tests := []struct {
			name, code, unitName string
		}{
			{"ferretería: bulto", "bulto", "Bulto"},
			{"almacén de ropa: par", "par", "Par"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				unit := e.addUnit(t, e.actor.TenantID, tt.code, tt.unitName)
				in := e.productInput(t)
				unitID := unit.ID()
				in.BaseUnitID = &unitID

				p, err := e.svc.CreateProduct(t.Context(), e.actor, in)
				require.NoError(t, err)

				require.Equal(t, unit.Ref(), p.BaseUnit())
				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Equal(t, unit.Ref(), stored.BaseUnit())
			})
		}
	})

	t.Run("crea un producto con el código de unidad deprecated", func(t *testing.T) {
		e := newEnv(t)
		in := e.productInput(t)
		in.BaseUnitID, in.BaseUnitCode = nil, "m"

		p, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.NoError(t, err)

		require.Equal(t, e.meters.Ref(), p.BaseUnit())
	})

	t.Run("cambia la unidad de un producto", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		piece := e.addUnit(t, e.actor.TenantID, "und", "Unidad")
		in := e.productInput(t)
		unitID := piece.ID()
		in.BaseUnitID = &unitID

		p, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)

		require.Equal(t, piece.Ref(), p.BaseUnit())
	})

	t.Run("rechaza un producto con una unidad que no existe en el tenant", func(t *testing.T) {
		otherTenant := uuid.Must(uuid.NewV7())
		tests := []struct {
			name   string
			mutate func(*env, *app.ProductInput)
		}{
			{"id inexistente", func(_ *env, in *app.ProductInput) {
				missing := uuid.Must(uuid.NewV7())
				in.BaseUnitID = &missing
			}},
			{"id de una unidad de otro tenant", func(e *env, in *app.ProductInput) {
				unitID := e.addUnit(t, otherTenant, "m", "Metro").ID()
				in.BaseUnitID = &unitID
			}},
			{"código inexistente", func(_ *env, in *app.ProductInput) { in.BaseUnitID, in.BaseUnitCode = nil, "docena" }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				in := e.productInput(t)
				tt.mutate(e, &in)

				_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
				require.ErrorIs(t, err, domain.ErrUnitNotFound)

				p := e.createProduct(t, e.productInput(t))
				_, err = e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
				require.ErrorIs(t, err, domain.ErrUnitNotFound)
			})
		}
	})

	t.Run("rechaza un producto sin unidad o con id y código a la vez", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*app.ProductInput)
		}{
			{"sin unidad", func(in *app.ProductInput) { in.BaseUnitID, in.BaseUnitCode = nil, "" }},
			{"id y código", func(in *app.ProductInput) { in.BaseUnitCode = "m" }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				in := e.productInput(t)
				tt.mutate(&in)

				_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
				require.ErrorIs(t, err, domain.ErrUnitRequired)
			})
		}
	})
}

func TestUnits(t *testing.T) {
	t.Run("crea, edita y lista unidades", func(t *testing.T) {
		e := newEnv(t)

		dozen, err := e.svc.CreateUnit(t.Context(), e.actor, "Docena", "Docena")
		require.NoError(t, err)
		require.Equal(t, "docena", dozen.Code())
		require.Equal(t, e.actor.TenantID, dozen.TenantID())
		require.Equal(t, testNow, dozen.CreatedAt())

		pair, err := e.svc.CreateUnit(t.Context(), e.actor, "par", "Par")
		require.NoError(t, err)
		pair, err = e.svc.UpdateUnit(t.Context(), e.actor, pair.ID(), "pares", "Pares")
		require.NoError(t, err)
		require.Equal(t, "pares", pair.Code())

		units, err := e.svc.ListUnits(t.Context(), e.actor)
		require.NoError(t, err)
		var names []string
		for _, u := range units {
			names = append(names, u.Name())
		}
		require.Equal(t, []string{"Docena", "Metro", "Pares"}, names)
	})

	t.Run("rechaza un código repetido en el tenant", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.CreateUnit(t.Context(), e.actor, " M ", "Metros")
		require.ErrorIs(t, err, domain.ErrUnitCodeTaken)

		kilo, err := e.svc.CreateUnit(t.Context(), e.actor, "kg", "Kilogramo")
		require.NoError(t, err)
		_, err = e.svc.UpdateUnit(t.Context(), e.actor, kilo.ID(), "m", "Kilogramo")
		require.ErrorIs(t, err, domain.ErrUnitCodeTaken)
	})

	t.Run("rechaza datos inválidos sin guardar", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.CreateUnit(t.Context(), e.actor, "caja x12", "Caja")
		require.ErrorIs(t, err, domain.ErrInvalidUnitCode)
		_, err = e.svc.UpdateUnit(t.Context(), e.actor, e.meters.ID(), "m", "")
		require.ErrorIs(t, err, domain.ErrInvalidUnitName)

		units, err := e.svc.ListUnits(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, units, 1)
		require.Equal(t, "Metro", units[0].Name())
	})

	t.Run("no se puede borrar una unidad en uso", func(t *testing.T) {
		e := newEnv(t)
		e.createProduct(t, e.productInput(t))

		err := e.svc.DeleteUnit(t.Context(), e.actor, e.meters.ID())
		require.ErrorIs(t, err, domain.ErrUnitInUse)
	})

	t.Run("borra una unidad sin productos", func(t *testing.T) {
		e := newEnv(t)

		require.NoError(t, e.svc.DeleteUnit(t.Context(), e.actor, e.meters.ID()))

		units, err := e.svc.ListUnits(t.Context(), e.actor)
		require.NoError(t, err)
		require.Empty(t, units)
	})

	t.Run("no ve, edita ni borra las unidades de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

		units, err := e.svc.ListUnits(t.Context(), other)
		require.NoError(t, err)
		require.Empty(t, units)
		_, err = e.svc.UpdateUnit(t.Context(), other, e.meters.ID(), "mt", "Metro")
		require.ErrorIs(t, err, domain.ErrUnitNotFound)
		require.ErrorIs(t, e.svc.DeleteUnit(t.Context(), other, e.meters.ID()), domain.ErrUnitNotFound)
	})

	t.Run("abre un span por caso de uso con el id de la unidad", func(t *testing.T) {
		e := newEnv(t)
		u, err := e.svc.CreateUnit(t.Context(), e.actor, "par", "Par")
		require.NoError(t, err)
		_, err = e.svc.UpdateUnit(t.Context(), e.actor, u.ID(), "par", "Pares")
		require.NoError(t, err)
		_, err = e.svc.ListUnits(t.Context(), e.actor)
		require.NoError(t, err)
		require.ErrorIs(t, e.svc.DeleteUnit(t.Context(), e.actor, uuid.Must(uuid.NewV7())), domain.ErrUnitNotFound)

		for _, name := range []string{"catalog.CreateUnit", "catalog.UpdateUnit"} {
			_, attrs := e.span(t, name)
			require.Equal(t, u.ID().String(), attrs["unit.id"].AsString(), name)
			require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString(), name)
		}
		e.span(t, "catalog.ListUnits")
		span, _ := e.span(t, "catalog.DeleteUnit")
		require.Equal(t, codes.Error, span.Status().Code)
	})
}

func TestSeedDefaultUnits(t *testing.T) {
	t.Run("siembra las unidades por defecto en un tenant nuevo", func(t *testing.T) {
		e := newEnv(t)
		tenantID := uuid.Must(uuid.NewV7())

		require.NoError(t, e.svc.SeedDefaultUnits(t.Context(), tenantID))

		units, err := e.svc.ListUnits(t.Context(), app.Actor{TenantID: tenantID})
		require.NoError(t, err)
		got := map[string]string{}
		for _, u := range units {
			got[u.Code()] = u.Name()
		}
		require.Equal(t, map[string]string{
			"und": "Unidad", "m": "Metro", "kg": "Kilogramo", "l": "Litro",
			"caja": "Caja", "rollo": "Rollo", "bulto": "Bulto", "galon": "Galón",
		}, got)

		_, attrs := e.span(t, "catalog.SeedDefaultUnits")
		require.Equal(t, attribute.StringValue(tenantID.String()), attrs["tenant.id"])
	})

	t.Run("sembrar dos veces el mismo tenant falla sin guardar nada más", func(t *testing.T) {
		e := newEnv(t)

		err := e.svc.SeedDefaultUnits(t.Context(), e.actor.TenantID)
		require.ErrorIs(t, err, domain.ErrUnitCodeTaken)

		units, err := e.svc.ListUnits(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, units, 1)
	})
}
