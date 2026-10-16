package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestVariantAxes(t *testing.T) {
	t.Run("crea un eje, lo lista por nombre y registra su span", func(t *testing.T) {
		e := newEnv(t)

		a, err := e.svc.CreateVariantAxis(t.Context(), e.actor, " Material ", []string{"Algodón", "Lino"})
		require.NoError(t, err)

		require.Equal(t, "material", a.Name())
		list, err := e.svc.ListVariantAxes(t.Context(), e.actor)
		require.NoError(t, err)
		names := make([]string, 0, len(list))
		for _, axis := range list {
			names = append(names, axis.Name())
		}
		require.Equal(t, []string{"color", "material", "presentación", "talla"}, names)
		_, attrs := e.span(t, "catalog.CreateVariantAxis")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, a.ID().String(), attrs["axis.id"].AsString())
	})

	t.Run("rechaza un nombre que ya usa otro eje del tenant", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.CreateVariantAxis(t.Context(), e.actor, "Talla", []string{"XS"})
		require.ErrorIs(t, err, domain.ErrAxisNameTaken)
	})

	t.Run("no se puede quitar un valor que usa una variante, ni con otras mayúsculas", func(t *testing.T) {
		e := newEnv(t)
		e.productWithAxes(t, map[string]string{"talla": "M"})
		talla := e.axis(t, "talla")

		updated, err := e.svc.UpdateVariantAxis(t.Context(), e.actor, talla.ID(), []string{"M", "L", "XL"})
		require.NoError(t, err, "quitar un valor sin uso sí se puede")
		require.Equal(t, []string{"M", "L", "XL"}, updated.Values())
		_, attrs := e.span(t, "catalog.UpdateVariantAxis")
		require.Equal(t, talla.ID().String(), attrs["axis.id"].AsString())

		_, err = e.svc.UpdateVariantAxis(t.Context(), e.actor, talla.ID(), []string{"m", "L"})
		require.ErrorIs(t, err, domain.ErrAxisValueInUse, "cambiar las mayúsculas de un valor en uso es quitarlo")
		_, err = e.svc.UpdateVariantAxis(t.Context(), e.actor, talla.ID(), []string{"L", "XL"})
		require.ErrorIs(t, err, domain.ErrAxisValueInUse)
	})

	t.Run("no se puede borrar un eje que usa una variante, y sí uno sin uso", func(t *testing.T) {
		e := newEnv(t)
		e.productWithAxes(t, map[string]string{"color": "Rojo"})

		require.ErrorIs(t, e.svc.DeleteVariantAxis(t.Context(), e.actor, e.axis(t, "color").ID()), domain.ErrAxisInUse)
		require.NoError(t, e.svc.DeleteVariantAxis(t.Context(), e.actor, e.axis(t, "talla").ID()))
	})

	t.Run("un eje de otro tenant no se encuentra", func(t *testing.T) {
		e := newEnv(t)
		other := e.addAxis(t, uuid.Must(uuid.NewV7()), "material", "Lino")

		_, err := e.svc.UpdateVariantAxis(t.Context(), e.actor, other.ID(), []string{"Seda"})
		require.ErrorIs(t, err, domain.ErrAxisNotFound)
		require.ErrorIs(t, e.svc.DeleteVariantAxis(t.Context(), e.actor, other.ID()), domain.ErrAxisNotFound)
	})
}

func TestVariantAxisValues(t *testing.T) {
	t.Run("agregar una variante escribe el valor como lo define el eje", func(t *testing.T) {
		tests := []struct {
			name          string
			first, second map[string]string
			want          map[string]string
		}{
			{"almacén de ropa: 'm' es la talla 'M'", map[string]string{"talla": "S"}, map[string]string{"Talla": " m "}, map[string]string{"talla": "M"}},
			{"ferretería: 'cuarto' es la presentación 'Cuarto'", map[string]string{"presentación": "Galón"}, map[string]string{"presentación": "cuarto"}, map[string]string{"presentación": "Cuarto"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				p := e.productWithAxes(t, tt.first)

				p, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("VAR-2", tt.second))
				require.NoError(t, err)
				require.Equal(t, tt.want, p.Variants()[1].AxisValues())
			})
		}
	})

	t.Run("una variante con la misma talla en otras mayúsculas es repetida", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "M"})

		_, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("VAR-2", map[string]string{"talla": "m"}))
		require.ErrorIs(t, err, domain.ErrDuplicateVariant)
	})

	t.Run("agregar una variante con un valor fuera del eje falla y no la guarda", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "S"})

		_, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("VAR-2", map[string]string{"talla": "XXL"}))
		require.ErrorIs(t, err, domain.ErrAxisValueNotAllowed)

		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Len(t, stored.Variants(), 1)
	})

	t.Run("editar una variante con un eje que el tenant no definió falla", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		d := p.DefaultVariant()

		_, err := e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), d.ID(), variantUpdate(d, map[string]string{"material": "Lino"}))
		require.ErrorIs(t, err, domain.ErrUnknownAxis)
	})

	t.Run("un eje de otro tenant no sirve", func(t *testing.T) {
		e := newEnv(t)
		e.addAxis(t, uuid.Must(uuid.NewV7()), "material", "Lino")
		p := e.createProduct(t, e.productInput(t))
		d := p.DefaultVariant()

		_, err := e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), d.ID(), variantUpdate(d, map[string]string{"material": "Lino"}))
		require.ErrorIs(t, err, domain.ErrUnknownAxis)
	})
}

// axis returns the actor's tenant axis named name.
func (e *env) axis(t *testing.T, name string) *domain.VariantAxis {
	t.Helper()
	list, err := e.svc.ListVariantAxes(t.Context(), e.actor)
	require.NoError(t, err)
	for _, a := range list {
		if a.Name() == name {
			return a
		}
	}
	t.Fatalf("no axis named %q", name)
	return nil
}

// variantUpdate keeps v as it is but for its axes.
func variantUpdate(v *domain.ProductVariant, axes map[string]string) app.VariantUpdate {
	return app.VariantUpdate{
		VariantInput: app.VariantInput{SKU: v.SKU(), Barcode: v.Barcode(), Cost: v.Cost(), Price: v.Price(), AxisValues: axes},
		Active:       v.IsActive(),
	}
}
