package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func productInput(t *testing.T) app.ProductInput {
	t.Helper()
	rate, err := domain.ParseTaxRate(dec("0.19"))
	require.NoError(t, err)
	return app.ProductInput{
		SKU:      "tub-pvc-12",
		Name:     "Tubo PVC presión 1/2 pulgada",
		BaseUnit: domain.UnitMeter,
		Cost:     dec("8333.33"),
		Price:    dec("10000"),
		TaxRate:  rate,
	}
}

func (e *env) createProduct(t *testing.T, in app.ProductInput) *domain.Product {
	t.Helper()
	p, err := e.svc.CreateProduct(t.Context(), e.actor, in)
	require.NoError(t, err)
	return p
}

func (e *env) createCategory(t *testing.T, name string, parentID *uuid.UUID) *domain.Category {
	t.Helper()
	c, err := e.svc.CreateCategory(t.Context(), e.actor, name, parentID)
	require.NoError(t, err)
	return c
}

func TestNewService(t *testing.T) {
	t.Run("falla si falta un repositorio", func(t *testing.T) {
		_, err := app.NewService(app.Deps{})
		require.Error(t, err)
	})
}

func TestCreateProduct(t *testing.T) {
	t.Run("crea un producto activo", func(t *testing.T) {
		e := newEnv(t)
		category := e.createCategory(t, "Plomería", nil)
		in := productInput(t)
		categoryID := category.ID()
		in.CategoryID = &categoryID

		p, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.NoError(t, err)

		require.True(t, p.IsActive())
		require.Equal(t, e.actor.TenantID, p.TenantID())
		require.Equal(t, "TUB-PVC-12", p.SKU())
		require.Equal(t, testNow, p.CreatedAt())
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, &categoryID, stored.CategoryID())
	})

	t.Run("rechaza producto con categoría inexistente", func(t *testing.T) {
		e := newEnv(t)
		in := productInput(t)
		missing := uuid.Must(uuid.NewV7())
		in.CategoryID = &missing

		_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza producto con una categoría de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		category := e.createCategory(t, "Plomería", nil)
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}
		in := productInput(t)
		categoryID := category.ID()
		in.CategoryID = &categoryID

		_, err := e.svc.CreateProduct(t.Context(), other, in)
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza producto con datos inválidos", func(t *testing.T) {
		e := newEnv(t)
		in := productInput(t)
		in.Price = dec("0")

		_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrInvalidPrice)
	})

	t.Run("rechaza SKU duplicado", func(t *testing.T) {
		e := newEnv(t)
		e.createProduct(t, productInput(t))

		_, err := e.svc.CreateProduct(t.Context(), e.actor, productInput(t))
		require.ErrorIs(t, err, domain.ErrSKUTaken)
	})

	t.Run("cuenta productos creados y abre su span", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))

		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.products_created", "", ""))
		_, attrs := e.span(t, "catalog.CreateProduct")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, e.actor.UserID.String(), attrs["user.id"].AsString())
		require.Equal(t, p.ID().String(), attrs["product.id"].AsString())
	})

	t.Run("registra el error en el span", func(t *testing.T) {
		e := newEnv(t)
		in := productInput(t)
		in.SKU = ""

		_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.Error(t, err)
		span, _ := e.span(t, "catalog.CreateProduct")
		require.Equal(t, codes.Error, span.Status().Code)
		require.Zero(t, e.counter(t, "cuadra.catalog.products_created", "", ""))
	})
}

func TestUpdateProduct(t *testing.T) {
	t.Run("actualiza datos y precio", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		in := productInput(t)
		in.Name, in.Price = "Tubo PVC presión 3/4 pulgada", dec("12500.50")

		updated, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)

		require.Equal(t, "Tubo PVC presión 3/4 pulgada", updated.Name())
		require.True(t, dec("12500.50").Equal(updated.Price()))
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.True(t, dec("12500.50").Equal(stored.Price()))
		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.price_changes", "", ""))
	})

	t.Run("no cuenta cambio de precio si el precio es el mismo", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		in := productInput(t)
		in.Price = dec("10000.00")

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)
		require.Zero(t, e.counter(t, "cuadra.catalog.price_changes", "", ""))
	})

	t.Run("rechaza cambio de precio de un producto inactivo", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		_, err := e.svc.DeactivateProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		in := productInput(t)
		in.Name, in.Price = "Otro nombre", dec("9000")

		_, err = e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.ErrorIs(t, err, domain.ErrProductInactive)
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", stored.Name(), "nothing is stored")
	})

	t.Run("un producto inactivo puede editar sus datos si no cambia el precio", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		_, err := e.svc.DeactivateProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		in := productInput(t)
		in.Name = "Otro nombre"

		updated, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)
		require.Equal(t, "Otro nombre", updated.Name())
	})

	t.Run("rechaza editar un producto de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

		_, err := e.svc.UpdateProduct(t.Context(), other, p.ID(), productInput(t))
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("rechaza producto con categoría inexistente", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))
		in := productInput(t)
		missing := uuid.Must(uuid.NewV7())
		in.CategoryID = &missing

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})
}

func TestProductActivation(t *testing.T) {
	t.Run("desactiva y activa un producto", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, productInput(t))

		off, err := e.svc.DeactivateProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.False(t, off.IsActive())
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.False(t, stored.IsActive())

		on, err := e.svc.ActivateProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.True(t, on.IsActive())
	})

	t.Run("rechaza desactivar un producto inexistente", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.DeactivateProduct(t.Context(), e.actor, uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

func TestSearchProducts(t *testing.T) {
	t.Run("busca en el tenant del actor y cuenta la búsqueda con resultados", func(t *testing.T) {
		e := newEnv(t)
		e.createProduct(t, productInput(t))

		page, err := e.svc.SearchProducts(t.Context(), e.actor, domain.ProductFilter{Query: "tubo pvc", Limit: 20})
		require.NoError(t, err)

		require.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)
		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.searches", "result", "hit"))
	})

	t.Run("cuenta la búsqueda sin resultados", func(t *testing.T) {
		e := newEnv(t)

		page, err := e.svc.SearchProducts(t.Context(), e.actor, domain.ProductFilter{Query: "martillo", Limit: 20})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.searches", "result", "empty"))
	})

	t.Run("acota limit y offset", func(t *testing.T) {
		e := newEnv(t)
		for _, f := range []domain.ProductFilter{{Limit: 500, Offset: -3}, {Limit: 0}} {
			_, err := e.svc.SearchProducts(t.Context(), e.actor, f)
			require.NoError(t, err)
		}

		require.Equal(t, 100, e.store.searches[0].Limit)
		require.Zero(t, e.store.searches[0].Offset)
		require.Equal(t, 20, e.store.searches[1].Limit)
	})

	t.Run("no registra el texto buscado en el span", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SearchProducts(t.Context(), e.actor, domain.ProductFilter{Query: "tubo pvc"})
		require.NoError(t, err)

		span, _ := e.span(t, "catalog.SearchProducts")
		for _, kv := range span.Attributes() {
			require.NotContains(t, kv.Value.String(), "tubo")
		}
	})
}

func TestCategories(t *testing.T) {
	t.Run("crea categorías y las lista por nombre", func(t *testing.T) {
		e := newEnv(t)
		plumbing := e.createCategory(t, "Plomería", nil)
		plumbingID := plumbing.ID()
		e.createCategory(t, "Eléctricos", nil)
		pipes := e.createCategory(t, "Tubería PVC", &plumbingID)

		list, err := e.svc.ListCategories(t.Context(), e.actor)
		require.NoError(t, err)

		require.Len(t, list, 3)
		require.Equal(t, "Eléctricos", list[0].Name())
		require.Equal(t, &plumbingID, pipes.ParentID())
	})

	t.Run("rechaza categoría con padre inexistente", func(t *testing.T) {
		e := newEnv(t)
		missing := uuid.Must(uuid.NewV7())

		_, err := e.svc.CreateCategory(t.Context(), e.actor, "Tubería PVC", &missing)
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("renombra y mueve una categoría a la raíz", func(t *testing.T) {
		e := newEnv(t)
		root := e.createCategory(t, "Plomería", nil)
		rootID := root.ID()
		child := e.createCategory(t, "Tuberia", &rootID)

		updated, err := e.svc.UpdateCategory(t.Context(), e.actor, child.ID(), "Tubería", nil)
		require.NoError(t, err)
		require.Equal(t, "Tubería", updated.Name())
		require.Nil(t, updated.ParentID())
	})

	t.Run("rechaza mover una categoría debajo de su propia descendiente", func(t *testing.T) {
		e := newEnv(t)
		a := e.createCategory(t, "A", nil)
		aID := a.ID()
		b := e.createCategory(t, "B", &aID)
		bID := b.ID()
		c := e.createCategory(t, "C", &bID)
		cID := c.ID()

		_, err := e.svc.UpdateCategory(t.Context(), e.actor, aID, "A", &cID)
		require.ErrorIs(t, err, domain.ErrCategoryCycle)
		stored, err := e.svc.ListCategories(t.Context(), e.actor)
		require.NoError(t, err)
		require.Nil(t, stored[0].ParentID(), "A stays at the root")
	})

	t.Run("rechaza que una categoría sea su propio padre", func(t *testing.T) {
		e := newEnv(t)
		a := e.createCategory(t, "A", nil)
		aID := a.ID()

		_, err := e.svc.UpdateCategory(t.Context(), e.actor, aID, "A", &aID)
		require.ErrorIs(t, err, domain.ErrInvalidCategoryParent)
	})

	t.Run("rechaza borrar una categoría con productos o subcategorías", func(t *testing.T) {
		e := newEnv(t)
		parent := e.createCategory(t, "Plomería", nil)
		parentID := parent.ID()
		e.createCategory(t, "Tubería", &parentID)
		withProduct := e.createCategory(t, "Grifería", nil)
		in := productInput(t)
		withProductID := withProduct.ID()
		in.CategoryID = &withProductID
		e.createProduct(t, in)

		require.ErrorIs(t, e.svc.DeleteCategory(t.Context(), e.actor, parentID), domain.ErrCategoryInUse)
		require.ErrorIs(t, e.svc.DeleteCategory(t.Context(), e.actor, withProductID), domain.ErrCategoryInUse)
	})

	t.Run("borra una categoría vacía", func(t *testing.T) {
		e := newEnv(t)
		c := e.createCategory(t, "Plomería", nil)

		require.NoError(t, e.svc.DeleteCategory(t.Context(), e.actor, c.ID()))
		list, err := e.svc.ListCategories(t.Context(), e.actor)
		require.NoError(t, err)
		require.Empty(t, list)
		_, attrs := e.span(t, "catalog.DeleteCategory")
		require.Equal(t, c.ID().String(), attrs["category.id"].AsString())
	})
}
