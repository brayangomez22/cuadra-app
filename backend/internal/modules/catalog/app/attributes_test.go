package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func materialInput() app.AttributeInput {
	return app.AttributeInput{Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Algodón"}}
}

func TestAttributeDefinitions(t *testing.T) {
	t.Run("crea una definición, la lista por etiqueta y registra su span", func(t *testing.T) {
		e := newEnv(t)

		d, err := e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.NoError(t, err)
		e.addAttribute(t, e.actor.TenantID, domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText})

		list, err := e.svc.ListAttributes(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Marca", list[0].Label())
		require.Equal(t, "Material", list[1].Label())
		_, attrs := e.span(t, "catalog.CreateAttribute")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
		require.Equal(t, d.ID().String(), attrs["attribute.id"].AsString())
	})

	t.Run("rechaza una clave que ya usa otra definición del tenant", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.NoError(t, err)

		_, err = e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.ErrorIs(t, err, domain.ErrAttributeKeyTaken)
	})

	t.Run("editar una definición conserva su clave y su tipo", func(t *testing.T) {
		e := newEnv(t)
		d, err := e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.NoError(t, err)

		updated, err := e.svc.UpdateAttribute(t.Context(), e.actor, d.ID(), app.AttributeUpdate{
			Label: "Material principal", Options: []string{"PVC", "Algodón", "Cobre"}, Required: true,
		})
		require.NoError(t, err)

		require.Equal(t, "material", updated.Key())
		require.Equal(t, domain.AttributeList, updated.Type())
		require.Equal(t, []string{"PVC", "Algodón", "Cobre"}, updated.Options())
		require.True(t, updated.Required())
		_, attrs := e.span(t, "catalog.UpdateAttribute")
		require.Equal(t, d.ID().String(), attrs["attribute.id"].AsString())
	})

	t.Run("no se puede quitar una opción que usa un producto", func(t *testing.T) {
		e := newEnv(t)
		d, err := e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.NoError(t, err)
		in := e.productInput(t)
		in.Attributes = map[string]domain.AttributeValue{"material": domain.TextValue("PVC")}
		e.createProduct(t, in)

		_, err = e.svc.UpdateAttribute(t.Context(), e.actor, d.ID(), app.AttributeUpdate{Label: "Material", Options: []string{"Algodón"}})
		require.ErrorIs(t, err, domain.ErrAttributeOptionInUse)

		_, err = e.svc.UpdateAttribute(t.Context(), e.actor, d.ID(), app.AttributeUpdate{Label: "Material", Options: []string{"PVC"}})
		require.NoError(t, err, "quitar una opción sin uso sí se puede")
	})

	t.Run("no se puede borrar un atributo que usa un producto, y sí uno sin uso", func(t *testing.T) {
		e := newEnv(t)
		used, err := e.svc.CreateAttribute(t.Context(), e.actor, materialInput())
		require.NoError(t, err)
		unused, err := e.svc.CreateAttribute(t.Context(), e.actor, app.AttributeInput{Key: "marca", Label: "Marca", Type: domain.AttributeText})
		require.NoError(t, err)
		in := e.productInput(t)
		in.Attributes = map[string]domain.AttributeValue{"material": domain.TextValue("PVC")}
		e.createProduct(t, in)

		require.ErrorIs(t, e.svc.DeleteAttribute(t.Context(), e.actor, used.ID()), domain.ErrAttributeInUse)
		require.NoError(t, e.svc.DeleteAttribute(t.Context(), e.actor, unused.ID()))

		list, err := e.svc.ListAttributes(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, list, 1)
	})

	t.Run("una definición de otro tenant no se encuentra", func(t *testing.T) {
		e := newEnv(t)
		d := e.addAttribute(t, uuid.Must(uuid.NewV7()), domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText})

		_, err := e.svc.UpdateAttribute(t.Context(), e.actor, d.ID(), app.AttributeUpdate{Label: "Otra"})
		require.ErrorIs(t, err, domain.ErrAttributeNotFound)
		require.ErrorIs(t, e.svc.DeleteAttribute(t.Context(), e.actor, d.ID()), domain.ErrAttributeNotFound)
		list, err := e.svc.ListAttributes(t.Context(), e.actor)
		require.NoError(t, err)
		require.Empty(t, list)
	})
}

func TestProductAttributes(t *testing.T) {
	tests := []struct {
		name  string
		def   domain.AttributeParams
		value domain.AttributeValue
		want  domain.AttributeValue
	}{
		{"ferretería: material de lista", domain.AttributeParams{
			Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Cobre"}, Required: true,
		}, domain.TextValue("pvc"), domain.TextValue("PVC")},
		{"almacén de ropa: marca de texto", domain.AttributeParams{
			Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true,
		}, domain.TextValue(" Textiles "), domain.TextValue("Textiles")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("crea un producto con sus atributos validados", func(t *testing.T) {
				e := newEnv(t)
				e.addAttribute(t, e.actor.TenantID, tt.def)
				in := e.productInput(t)
				in.Attributes = map[string]domain.AttributeValue{tt.def.Key: tt.value}

				p := e.createProduct(t, in)

				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Equal(t, map[string]domain.AttributeValue{tt.def.Key: tt.want}, stored.Attributes().Values())
			})

			t.Run("rechaza un producto sin un atributo obligatorio y no lo guarda", func(t *testing.T) {
				e := newEnv(t)
				e.addAttribute(t, e.actor.TenantID, tt.def)

				_, err := e.svc.CreateProduct(t.Context(), e.actor, e.productInput(t))
				require.ErrorIs(t, err, domain.ErrAttributeRequired)
				page, err := e.svc.SearchProducts(t.Context(), e.actor, domain.ProductFilter{})
				require.NoError(t, err)
				require.Zero(t, page.Total)
			})
		})
	}

	t.Run("actualizar sin atributos conserva los actuales", func(t *testing.T) {
		e := newEnv(t)
		e.addAttribute(t, e.actor.TenantID, domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true})
		in := e.productInput(t)
		in.Attributes = map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")}
		p := e.createProduct(t, in)
		in.Attributes = nil
		in.Name = "Tubo PVC presión 3/4 pulgada"

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)

		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")}, stored.Attributes().Values())
	})

	t.Run("actualizar con atributos los valida y los reemplaza", func(t *testing.T) {
		e := newEnv(t)
		e.addAttribute(t, e.actor.TenantID, domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText})
		e.addAttribute(t, e.actor.TenantID, domain.AttributeParams{Key: "importado", Label: "Importado", Type: domain.AttributeBoolean})
		in := e.productInput(t)
		in.Attributes = map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")}
		p := e.createProduct(t, in)

		in.Attributes = map[string]domain.AttributeValue{"importado": domain.TextValue("sí")}
		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.ErrorIs(t, err, domain.ErrInvalidAttributeValue)

		in.Attributes = map[string]domain.AttributeValue{"importado": domain.BoolValue(true)}
		_, err = e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)
		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, map[string]domain.AttributeValue{"importado": domain.BoolValue(true)}, stored.Attributes().Values())
	})

	t.Run("las definiciones de otro tenant no aplican", func(t *testing.T) {
		e := newEnv(t)
		e.addAttribute(t, uuid.Must(uuid.NewV7()), domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true})
		in := e.productInput(t)

		_, err := e.svc.CreateProduct(t.Context(), e.actor, in)
		require.NoError(t, err, "el atributo obligatorio es de otro tenant")

		in.SKU = "OTRO"
		in.Attributes = map[string]domain.AttributeValue{"marca": domain.TextValue("Acme")}
		_, err = e.svc.CreateProduct(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrUnknownAttribute)
	})
}
