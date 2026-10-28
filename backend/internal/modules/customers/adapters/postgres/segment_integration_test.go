//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func segmentNames(t *testing.T, list []*domain.Segment) []string {
	t.Helper()
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.Name())
	}
	return out
}

func TestSegmentRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("crea, renombra y lista los segmentos por nombre", func(t *testing.T) {
		tenant := f.createTenant(t)
		wholesale := f.createSegment(t, tenant, "Mayorista")
		f.createSegment(t, tenant, "Convenio EPS")
		require.NoError(t, wholesale.Rename("al por mayor"))
		require.NoError(t, f.segments.Update(t.Context(), wholesale))

		list, err := f.segments.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"al por mayor", "Convenio EPS"}, segmentNames(t, list))
		got, err := f.segments.GetByID(t.Context(), tenant, wholesale.ID())
		require.NoError(t, err)
		require.Equal(t, wholesale, got)
	})

	t.Run("no permite dos segmentos con el mismo nombre sin importar mayúsculas", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createSegment(t, tenant, "Maestro de obra")
		dup, err := domain.NewSegment(tenant, "MAESTRO DE OBRA", now)
		require.NoError(t, err)
		require.ErrorIs(t, f.segments.Create(t.Context(), dup), domain.ErrSegmentNameTaken)

		other := f.createSegment(t, tenant, "Contratista")
		require.NoError(t, other.Rename("maestro de obra"))
		require.ErrorIs(t, f.segments.Update(t.Context(), other), domain.ErrSegmentNameTaken)
	})

	t.Run("no borra un segmento con clientes, aunque estén inactivos", func(t *testing.T) {
		tenant := f.createTenant(t)
		segment := f.createSegment(t, tenant, "Mayorista")
		id := segment.ID()
		p := person("1020304050", "Carlos Pérez")
		p.SegmentID, p.Active = &id, false
		f.createCustomer(t, tenant, p)

		require.ErrorIs(t, f.segments.Delete(t.Context(), tenant, segment.ID()), domain.ErrSegmentInUse)
	})

	t.Run("borra un segmento sin clientes; uno inexistente no se encuentra", func(t *testing.T) {
		tenant := f.createTenant(t)
		segment := f.createSegment(t, tenant, "Mayorista")

		require.NoError(t, f.segments.Delete(t.Context(), tenant, segment.ID()))
		require.ErrorIs(t, f.segments.Delete(t.Context(), tenant, segment.ID()), domain.ErrSegmentNotFound)
		_, err := f.segments.GetByID(t.Context(), tenant, segment.ID())
		require.ErrorIs(t, err, domain.ErrSegmentNotFound)
	})

	t.Run("aislamiento: el tenant B no ve, renombra ni borra los segmentos del A", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		segment := f.createSegment(t, tenantA, "Mayorista")

		list, err := f.segments.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Empty(t, list)
		_, err = f.segments.GetByID(t.Context(), tenantB, segment.ID())
		require.ErrorIs(t, err, domain.ErrSegmentNotFound)
		require.ErrorIs(t, f.segments.Delete(t.Context(), tenantB, segment.ID()), domain.ErrSegmentNotFound)
		hijack, err := domain.RehydrateSegment(domain.SegmentSnapshot{ID: segment.ID(), TenantID: tenantB, Name: "Robado", CreatedAt: now})
		require.NoError(t, err)
		require.ErrorIs(t, f.segments.Update(t.Context(), hijack), domain.ErrSegmentNotFound)

		got, err := f.segments.GetByID(t.Context(), tenantA, segment.ID())
		require.NoError(t, err)
		require.Equal(t, "Mayorista", got.Name())
		// Tenant B can use the same name: uniqueness is per tenant.
		f.createSegment(t, tenantB, "Mayorista")
	})
}
