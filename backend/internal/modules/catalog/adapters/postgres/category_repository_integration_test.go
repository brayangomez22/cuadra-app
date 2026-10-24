//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestCategoryRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, lista por nombre y lee categorías", func(t *testing.T) {
		tenant := f.createTenant(t)
		plumbing := f.createCategory(t, tenant, "Plomería", nil)
		plumbingID := plumbing.ID()
		pipes := f.createCategory(t, tenant, "Tubería", &plumbingID)
		f.createCategory(t, tenant, "Eléctricos", nil)

		list, err := f.categories.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, list, 3)
		require.Equal(t, "Eléctricos", list[0].Name())

		got, err := f.categories.GetByID(t.Context(), tenant, pipes.ID())
		require.NoError(t, err)
		require.Equal(t, "Tubería", got.Name())
		require.Equal(t, &plumbingID, got.ParentID())
		require.Equal(t, now, got.CreatedAt())
	})

	t.Run("devuelve ErrCategoryNotFound si no existe", func(t *testing.T) {
		tenant := f.createTenant(t)
		_, err := f.categories.GetByID(t.Context(), tenant, uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza un padre inexistente o de otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		parentB := f.createCategory(t, tenantB, "Plomería", nil)
		for _, parentID := range []uuid.UUID{uuid.Must(uuid.NewV7()), parentB.ID()} {
			c, err := domain.NewCategory(tenantA, "Tubería", &parentID, now)
			require.NoError(t, err)
			require.ErrorIs(t, f.categories.Create(t.Context(), c), domain.ErrCategoryNotFound)
		}
	})

	t.Run("renombra y mueve una categoría", func(t *testing.T) {
		tenant := f.createTenant(t)
		a := f.createCategory(t, tenant, "A", nil)
		b := f.createCategory(t, tenant, "B", nil)
		aID := a.ID()
		require.NoError(t, b.Rename("B2"))
		require.NoError(t, b.MoveTo(&aID))

		require.NoError(t, f.categories.Update(t.Context(), b))

		got, err := f.categories.GetByID(t.Context(), tenant, b.ID())
		require.NoError(t, err)
		require.Equal(t, "B2", got.Name())
		require.Equal(t, &aID, got.ParentID())
	})

	t.Run("devuelve los ancestros del más cercano al más lejano", func(t *testing.T) {
		tenant := f.createTenant(t)
		a := f.createCategory(t, tenant, "A", nil)
		aID := a.ID()
		b := f.createCategory(t, tenant, "B", &aID)
		bID := b.ID()
		c := f.createCategory(t, tenant, "C", &bID)

		chain, err := f.categories.Ancestors(t.Context(), tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{c.ID(), bID, aID}, chain)

		none, err := f.categories.Ancestors(t.Context(), tenant, uuid.Must(uuid.NewV7()))
		require.NoError(t, err)
		require.Empty(t, none)
	})

	t.Run("rechaza borrar una categoría con productos o subcategorías", func(t *testing.T) {
		tenant := f.createTenant(t)
		parent := f.createCategory(t, tenant, "Plomería", nil)
		parentID := parent.ID()
		f.createCategory(t, tenant, "Tubería", &parentID)
		withProduct := f.createCategory(t, tenant, "Grifería", nil)
		f.createProduct(t, tenant, "GRI-1", "Grifo", withCategory(withProduct.ID()))

		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, parentID), domain.ErrCategoryInUse)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, withProduct.ID()), domain.ErrCategoryInUse)
	})

	t.Run("borra una categoría vacía", func(t *testing.T) {
		tenant := f.createTenant(t)
		c := f.createCategory(t, tenant, "Plomería", nil)

		require.NoError(t, f.categories.Delete(t.Context(), tenant, c.ID()))
		_, err := f.categories.GetByID(t.Context(), tenant, c.ID())
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, c.ID()), domain.ErrCategoryNotFound)
	})
}
