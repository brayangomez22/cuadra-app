package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func (s *stubUseCases) CreateAttribute(_ context.Context, a app.Actor, in app.AttributeInput) (*domain.AttributeDefinition, error) {
	s.actor, s.attrInput = a, in
	return s.attribute, s.err
}

func (s *stubUseCases) UpdateAttribute(_ context.Context, a app.Actor, id uuid.UUID, in app.AttributeUpdate) (*domain.AttributeDefinition, error) {
	s.actor, s.id, s.attrUpdate = a, id, in
	return s.attribute, s.err
}

func (s *stubUseCases) ListAttributes(_ context.Context, a app.Actor) ([]*domain.AttributeDefinition, error) {
	s.actor = a
	return s.attributes, s.err
}

func (s *stubUseCases) DeleteAttribute(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

func (s *stubUseCases) CreateVariantAxis(_ context.Context, a app.Actor, name string, values []string) (*domain.VariantAxis, error) {
	s.actor, s.name, s.values = a, name, values
	return s.axis, s.err
}

func (s *stubUseCases) UpdateVariantAxis(_ context.Context, a app.Actor, id uuid.UUID, values []string) (*domain.VariantAxis, error) {
	s.actor, s.id, s.values = a, id, values
	return s.axis, s.err
}

func (s *stubUseCases) ListVariantAxes(_ context.Context, a app.Actor) ([]*domain.VariantAxis, error) {
	s.actor = a
	return s.axes, s.err
}

func (s *stubUseCases) DeleteVariantAxis(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

func mustAttribute(t *testing.T, p domain.AttributeParams) *domain.AttributeDefinition {
	t.Helper()
	d, err := domain.NewAttributeDefinition(owner.TenantID, p, now)
	require.NoError(t, err)
	return d
}

// errorBody decodes the Error schema with its details.
func errorBody(t *testing.T, body []byte) (code, message string, details map[string]string) {
	t.Helper()
	var e struct {
		Error struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Details map[string]string `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &e))
	return e.Error.Code, e.Error.Message, e.Error.Details
}

func TestProductAttributesHTTP(t *testing.T) {
	t.Run("crear un producto pasa los atributos de texto y sí/no al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}
		body := `{"sku":"tub-pvc-12","name":"Tubo","base_unit_id":"` + meters.ID().String() + `","cost":"1","price":"2","tax_rate":"0.19",` +
			`"attributes":{"marca":"Acme","hojas":"2.5","importado":true}}`

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, map[string]domain.AttributeValue{
			"marca": domain.TextValue("Acme"), "hojas": domain.TextValue("2.5"), "importado": domain.BoolValue(true),
		}, uc.input.Attributes)
	})

	t.Run("un valor que no es texto ni sí/no se rechaza nombrando el atributo", func(t *testing.T) {
		uc := &stubUseCases{product: product(t, "8333.33", "10000")}
		body := `{"sku":"tub-pvc-12","name":"Tubo","base_unit_id":"` + meters.ID().String() + `","cost":"1","price":"2","tax_rate":"0.19","attributes":{"hojas":100}}`

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", body)

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		code, _, details := errorBody(t, rec.Body.Bytes())
		require.Equal(t, "invalid_attribute_value", code)
		require.Equal(t, map[string]string{"attribute": "hojas"}, details)
	})

	t.Run("la respuesta incluye los atributos del producto, o {} si no tiene", func(t *testing.T) {
		p := product(t, "8333.33", "10000")
		attrs, err := domain.RehydrateAttributes(map[string]domain.AttributeValue{
			"marca": domain.TextValue("Acme"), "importado": domain.BoolValue(false),
		})
		require.NoError(t, err)
		require.NoError(t, p.UpdateDetails(domain.ProductDetails{Name: p.Name(), BaseUnit: p.BaseUnit(), TaxRate: p.TaxRate(), Attributes: attrs}))

		rec := serve(t, &stubUseCases{product: p}, owner, http.MethodGet, "/api/v1/products/"+p.ID().String(), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, map[string]any{"marca": "Acme", "importado": false}, decodeMap(t, rec)["attributes"])

		rec = serve(t, &stubUseCases{product: product(t, "1", "2")}, owner, http.MethodGet, "/api/v1/products/"+p.ID().String(), "")
		require.Equal(t, map[string]any{}, decodeMap(t, rec)["attributes"])
	})

	t.Run("un error de atributo responde 422 con su etiqueta en el mensaje y su clave en details", func(t *testing.T) {
		tests := []struct {
			name, code string
			err        error
			wantInMsg  string
		}{
			{"obligatorio", "attribute_required",
				&domain.AttributeError{Key: "marca", Label: "Marca", Err: domain.ErrAttributeRequired}, "«Marca»"},
			{"valor inválido", "invalid_attribute_value",
				&domain.AttributeError{Key: "material", Label: "Material", Err: domain.ErrInvalidAttributeValue}, "«Material»"},
			{"no definido: se nombra por su clave", "unknown_attribute",
				&domain.AttributeError{Key: "talla", Err: domain.ErrUnknownAttribute}, "«talla»"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				uc := &stubUseCases{err: tt.err}

				rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products", productBody)

				require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
				code, message, details := errorBody(t, rec.Body.Bytes())
				require.Equal(t, tt.code, code)
				require.Contains(t, message, tt.wantInMsg)
				var attrErr *domain.AttributeError
				require.ErrorAs(t, tt.err, &attrErr)
				require.Equal(t, map[string]string{"attribute": attrErr.Key}, details)
			})
		}
	})

	t.Run("un eje que no admite el valor responde 422 con el eje en details", func(t *testing.T) {
		uc := &stubUseCases{err: &domain.AxisError{Axis: "talla", Err: domain.ErrAxisValueNotAllowed}}
		body := `{"sku":"cam-xl","cost":"1","price":"2","axis_values":{"talla":"XXL"}}`

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/products/"+uuid.Must(uuid.NewV7()).String()+"/variants", body)

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		code, message, details := errorBody(t, rec.Body.Bytes())
		require.Equal(t, "axis_value_not_allowed", code)
		require.Contains(t, message, "«talla»")
		require.Equal(t, map[string]string{"axis": "talla"}, details)
	})
}

func TestAttributeDefinitionsHTTP(t *testing.T) {
	t.Run("crear una definición traduce el tipo y las opciones y responde 201", func(t *testing.T) {
		d := mustAttribute(t, domain.AttributeParams{Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Algodón"}})
		uc := &stubUseCases{attribute: d}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"material","label":"Material","type":"list","options":["PVC","Algodón"],"required":true}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.AttributeInput{
			Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Algodón"}, Required: true,
		}, uc.attrInput)
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		body := decodeMap(t, rec)
		require.Equal(t, "list", body["type"])
		require.Equal(t, []any{"PVC", "Algodón"}, body["options"])
	})

	t.Run("una definición sin opciones las devuelve vacías", func(t *testing.T) {
		d := mustAttribute(t, domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText})

		rec := serve(t, &stubUseCases{attributes: []*domain.AttributeDefinition{d}}, cashier(), http.MethodGet, "/api/v1/product-attributes", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"options":[]`)
	})

	t.Run("un tipo desconocido responde 422 invalid_attribute_type", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, owner, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"vence","label":"Vence","type":"date","required":false}`)

		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_attribute_type")
	})

	t.Run("una definición inexistente responde 404 y una en uso 409", func(t *testing.T) {
		path := "/api/v1/product-attributes/" + uuid.Must(uuid.NewV7()).String()

		requireError(t, serve(t, &stubUseCases{err: domain.ErrAttributeNotFound}, owner, http.MethodDelete, path, ""),
			http.StatusNotFound, "attribute_not_found")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrAttributeInUse}, owner, http.MethodDelete, path, ""),
			http.StatusConflict, "attribute_in_use")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrAttributeOptionInUse}, owner, http.MethodPut, path,
			`{"label":"Material","options":["PVC"],"required":false}`), http.StatusConflict, "attribute_option_in_use")
	})
}

func TestVariantAxesHTTP(t *testing.T) {
	t.Run("crear un eje responde 201 con su nombre y sus valores", func(t *testing.T) {
		a, err := domain.NewVariantAxis(owner.TenantID, "color", []string{"Rojo", "Azul"}, now)
		require.NoError(t, err)
		uc := &stubUseCases{axis: a}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/variant-axes", `{"name":"Color","values":["Rojo","Azul"]}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "Color", uc.name)
		require.Equal(t, []string{"Rojo", "Azul"}, uc.values)
		body := decodeMap(t, rec)
		require.Equal(t, "color", body["name"])
		require.Equal(t, []any{"Rojo", "Azul"}, body["values"])
	})

	t.Run("un eje inexistente responde 404 y un valor en uso 409", func(t *testing.T) {
		path := "/api/v1/variant-axes/" + uuid.Must(uuid.NewV7()).String()

		requireError(t, serve(t, &stubUseCases{err: domain.ErrAxisNotFound}, owner, http.MethodPut, path, `{"values":["M"]}`),
			http.StatusNotFound, "axis_not_found")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrAxisValueInUse}, owner, http.MethodPut, path, `{"values":["M"]}`),
			http.StatusConflict, "axis_value_in_use")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrAxisInUse}, owner, http.MethodDelete, path, ""),
			http.StatusConflict, "axis_in_use")
	})
}
