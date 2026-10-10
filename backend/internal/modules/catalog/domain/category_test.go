package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestNewCategory(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())

	t.Run("crea categoría raíz", func(t *testing.T) {
		category, err := domain.NewCategory(tenantID, "  Plomería  ", nil, testNow)
		require.NoError(t, err)
		require.Equal(t, 7, int(category.ID().Version()))
		require.Equal(t, tenantID, category.TenantID())
		require.Equal(t, "Plomería", category.Name())
		require.Nil(t, category.ParentID())
		require.Equal(t, testNow.UTC(), category.CreatedAt())
	})

	t.Run("crea subcategoría con padre", func(t *testing.T) {
		parent, err := domain.NewCategory(tenantID, "Plomería", nil, testNow)
		require.NoError(t, err)
		parentID := parent.ID()

		child, err := domain.NewCategory(tenantID, "Tubería PVC", &parentID, testNow)
		require.NoError(t, err)
		require.Equal(t, &parentID, child.ParentID())
	})

	t.Run("rechaza categoría con datos inválidos", func(t *testing.T) {
		nilParent := uuid.Nil
		_, err := domain.NewCategory(uuid.Nil, "Plomería", nil, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)

		for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
			_, err := domain.NewCategory(tenantID, name, nil, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidCategoryName)
		}

		_, err = domain.NewCategory(tenantID, "Plomería", &nilParent, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidCategoryParent)
	})
}

func TestCategoryChanges(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())

	t.Run("renombra una categoría", func(t *testing.T) {
		category, err := domain.NewCategory(tenantID, "Plomeria", nil, testNow)
		require.NoError(t, err)

		require.NoError(t, category.Rename("  Plomería  "))
		require.Equal(t, "Plomería", category.Name())

		require.ErrorIs(t, category.Rename(" "), domain.ErrInvalidCategoryName)
		require.Equal(t, "Plomería", category.Name())
	})

	t.Run("mueve una categoría a otro padre o a la raíz", func(t *testing.T) {
		category, err := domain.NewCategory(tenantID, "Tubería PVC", nil, testNow)
		require.NoError(t, err)
		parentID := uuid.Must(uuid.NewV7())

		require.NoError(t, category.MoveTo(&parentID))
		require.Equal(t, &parentID, category.ParentID())

		require.NoError(t, category.MoveTo(nil))
		require.Nil(t, category.ParentID())
	})

	t.Run("rechaza categoría que es su propio padre", func(t *testing.T) {
		category, err := domain.NewCategory(tenantID, "Plomería", nil, testNow)
		require.NoError(t, err)
		selfID := category.ID()

		require.ErrorIs(t, category.MoveTo(&selfID), domain.ErrInvalidCategoryParent)
		require.Nil(t, category.ParentID())

		_, err = domain.RehydrateCategory(domain.CategorySnapshot{
			ID: selfID, TenantID: tenantID, Name: "Plomería", ParentID: &selfID, CreatedAt: testNow,
		})
		require.ErrorIs(t, err, domain.ErrInvalidCategoryParent)
	})

	t.Run("reconstruye una categoría guardada", func(t *testing.T) {
		id, parentID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		category, err := domain.RehydrateCategory(domain.CategorySnapshot{
			ID: id, TenantID: tenantID, Name: "Tubería PVC", ParentID: &parentID, CreatedAt: testNow,
		})
		require.NoError(t, err)
		require.Equal(t, id, category.ID())
		require.Equal(t, &parentID, category.ParentID())

		_, err = domain.RehydrateCategory(domain.CategorySnapshot{TenantID: tenantID, Name: "X", CreatedAt: testNow})
		require.ErrorIs(t, err, domain.ErrInvalidCategoryID)
	})
}
