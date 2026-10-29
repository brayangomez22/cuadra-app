//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func TestImportFiles(t *testing.T) {
	f := newFixture(t)
	later := now.Add(31 * 24 * time.Hour)

	t.Run("encuentra en todas las empresas las importaciones terminadas con archivo", func(t *testing.T) {
		tenantA, locationA := f.tenant(t)
		tenantB, locationB := f.tenant(t)
		rejectedA := f.stored(t, tenantA, locationA, domain.RowError{Row: 2, Code: "invalid_sku"})
		rejectedB := f.stored(t, tenantB, locationB, domain.RowError{Row: 2, Code: "invalid_sku"})
		f.stored(t, tenantA, locationA) // validated: it may still be confirmed

		refs, err := f.imports.PurgeableFiles(t.Context(), later, 100)
		require.NoError(t, err)
		require.Subset(t, refs, []domain.FileRef{
			{TenantID: tenantA, ImportID: rejectedA.ID()}, {TenantID: tenantB, ImportID: rejectedB.ID()},
		})
		for _, ref := range refs {
			imp, err := f.imports.Get(t.Context(), ref.TenantID, ref.ImportID)
			require.NoError(t, err)
			require.Equal(t, domain.StatusRejected, imp.Status())
		}

		recent, err := f.imports.PurgeableFiles(t.Context(), now, 100)
		require.NoError(t, err)
		require.NotContains(t, recent, domain.FileRef{TenantID: tenantA, ImportID: rejectedA.ID()})
	})

	t.Run("borra el archivo y conserva la importación con sus errores", func(t *testing.T) {
		tenant, location := f.tenant(t)
		imp := f.stored(t, tenant, location, domain.RowError{Row: 2, Code: "invalid_sku"})

		require.NoError(t, f.imports.DropFile(t.Context(), tenant, imp.ID()))
		file, err := f.imports.File(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Empty(t, file)
		_, total, err := f.imports.Errors(t.Context(), tenant, imp.ID(), 10, 0)
		require.NoError(t, err)
		require.Equal(t, 1, total)

		refs, err := f.imports.PurgeableFiles(t.Context(), later, 100)
		require.NoError(t, err)
		require.NotContains(t, refs, domain.FileRef{TenantID: tenant, ImportID: imp.ID()}, "a dropped file is not found again")
		require.ErrorIs(t, f.imports.DropFile(t.Context(), tenant, imp.ID()), domain.ErrImportNotFound)
	})

	t.Run("aislamiento: otra empresa no borra el archivo de la importación del A", func(t *testing.T) {
		tenantA, locationA := f.tenant(t)
		tenantB, _ := f.tenant(t)
		imp := f.stored(t, tenantA, locationA, domain.RowError{Row: 2, Code: "invalid_sku"})

		require.ErrorIs(t, f.imports.DropFile(t.Context(), tenantB, imp.ID()), domain.ErrImportNotFound)
		require.ErrorIs(t, f.imports.DropFile(t.Context(), uuid.Must(uuid.NewV7()), imp.ID()), domain.ErrImportNotFound)
		file, err := f.imports.File(t.Context(), tenantA, imp.ID())
		require.NoError(t, err)
		require.Equal(t, []byte("contenido"), file)
	})
}
