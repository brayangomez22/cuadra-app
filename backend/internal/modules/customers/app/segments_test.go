package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestSegments(t *testing.T) {
	t.Run("crea, renombra y lista los segmentos del tenant por nombre", func(t *testing.T) {
		e := newEnv(t)
		e.addSegment(t, newID(), "De otro tenant")
		wholesale, err := e.svc.CreateSegment(t.Context(), e.actor, " Mayorista ")
		require.NoError(t, err)
		_, err = e.svc.CreateSegment(t.Context(), e.actor, "Convenio EPS")
		require.NoError(t, err)

		renamed, err := e.svc.RenameSegment(t.Context(), e.actor, wholesale.ID(), "Al por mayor")
		require.NoError(t, err)
		require.Equal(t, "Al por mayor", renamed.Name())

		list, err := e.svc.ListSegments(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Al por mayor", list[0].Name())
		require.Equal(t, "Convenio EPS", list[1].Name())
		_, attrs := e.span(t, "customers.RenameSegment")
		require.Equal(t, wholesale.ID().String(), attrs["segment.id"].AsString())
	})

	t.Run("no permite dos segmentos con el mismo nombre sin importar mayúsculas", func(t *testing.T) {
		e := newEnv(t)
		e.addSegment(t, e.actor.TenantID, "Maestro de obra")
		_, err := e.svc.CreateSegment(t.Context(), e.actor, "MAESTRO DE OBRA")
		require.ErrorIs(t, err, domain.ErrSegmentNameTaken)
	})

	t.Run("no borra un segmento en uso, aunque sus clientes estén inactivos", func(t *testing.T) {
		e := newEnv(t)
		segment := e.addSegment(t, e.actor.TenantID, "Mayorista")
		p := person("1020304050", "Carlos Pérez")
		id := segment.ID()
		p.SegmentID, p.Active = &id, false
		e.addCustomer(t, p)

		require.ErrorIs(t, e.svc.DeleteSegment(t.Context(), e.actor, segment.ID()), domain.ErrSegmentInUse)
	})

	t.Run("borra un segmento sin clientes", func(t *testing.T) {
		e := newEnv(t)
		segment := e.addSegment(t, e.actor.TenantID, "Mayorista")
		require.NoError(t, e.svc.DeleteSegment(t.Context(), e.actor, segment.ID()))
		list, err := e.svc.ListSegments(t.Context(), e.actor)
		require.NoError(t, err)
		require.Empty(t, list)
	})
}

func TestSeedSegments(t *testing.T) {
	t.Run("siembra los segmentos de la plantilla de cada tipo de negocio", func(t *testing.T) {
		tests := []struct {
			name     string
			segments []string
		}{
			{"ferretería", []string{"Público", "Maestro de obra", "Contratista", "Empresa"}},
			{"almacén de ropa", []string{"Público", "Mayorista"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				require.NoError(t, e.svc.SeedSegments(t.Context(), e.actor.TenantID, tt.segments))

				list, err := e.svc.ListSegments(t.Context(), e.actor)
				require.NoError(t, err)
				names := make([]string, 0, len(list))
				for _, s := range list {
					names = append(names, s.Name())
				}
				require.ElementsMatch(t, tt.segments, names)
				_, attrs := e.span(t, "customers.SeedSegments")
				require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
			})
		}
	})
}
