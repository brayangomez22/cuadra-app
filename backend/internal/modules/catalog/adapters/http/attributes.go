package httpadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// AttributeUseCases are the use cases of the attribute definitions.
type AttributeUseCases interface {
	CreateAttribute(ctx context.Context, a app.Actor, in app.AttributeInput) (*domain.AttributeDefinition, error)
	UpdateAttribute(ctx context.Context, a app.Actor, id uuid.UUID, in app.AttributeUpdate) (*domain.AttributeDefinition, error)
	ListAttributes(ctx context.Context, a app.Actor) ([]*domain.AttributeDefinition, error)
	DeleteAttribute(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// attributeHandlers are the operations on attribute definitions.
type attributeHandlers struct{ uc AttributeUseCases }

// ListAttributes lists the tenant's attribute definitions.
func (h *attributeHandlers) ListAttributes(ctx context.Context, _ catalogapi.ListAttributesRequestObject) (catalogapi.ListAttributesResponseObject, error) {
	defs, err := h.uc.ListAttributes(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]catalogapi.AttributeDefinition, 0, len(defs))
	for _, d := range defs {
		items = append(items, attributeBody(d))
	}
	return catalogapi.ListAttributes200JSONResponse{Items: items}, nil
}

// CreateAttribute defines a product attribute.
func (h *attributeHandlers) CreateAttribute(ctx context.Context, req catalogapi.CreateAttributeRequestObject) (catalogapi.CreateAttributeResponseObject, error) {
	b := req.Body
	typ, err := domain.ParseAttributeType(string(b.Type))
	if err != nil {
		return nil, err
	}
	d, err := h.uc.CreateAttribute(ctx, actor(ctx), app.AttributeInput{
		Key: b.Key, Label: b.Label, Type: typ, Options: options(b.Options), Required: b.Required,
	})
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateAttribute201JSONResponse(attributeBody(d)), nil
}

// UpdateAttribute changes a definition's label, options and whether it is
// required.
func (h *attributeHandlers) UpdateAttribute(ctx context.Context, req catalogapi.UpdateAttributeRequestObject) (catalogapi.UpdateAttributeResponseObject, error) {
	b := req.Body
	d, err := h.uc.UpdateAttribute(ctx, actor(ctx), req.AttributeId, app.AttributeUpdate{
		Label: b.Label, Options: options(b.Options), Required: b.Required,
	})
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateAttribute200JSONResponse(attributeBody(d)), nil
}

// DeleteAttribute removes a definition no product has.
func (h *attributeHandlers) DeleteAttribute(ctx context.Context, req catalogapi.DeleteAttributeRequestObject) (catalogapi.DeleteAttributeResponseObject, error) {
	if err := h.uc.DeleteAttribute(ctx, actor(ctx), req.AttributeId); err != nil {
		return nil, err
	}
	return catalogapi.DeleteAttribute204Response{}, nil
}

func options(o *catalogapi.AttributeOptions) []string {
	if o == nil {
		return nil
	}
	return *o
}

func attributeBody(d *domain.AttributeDefinition) catalogapi.AttributeDefinition {
	opts := d.Options()
	if opts == nil {
		opts = []string{}
	}
	return catalogapi.AttributeDefinition{
		Id: d.ID(), Key: d.Key(), Label: d.Label(), Type: catalogapi.AttributeType(d.Type()),
		Options: opts, Required: d.Required(),
	}
}

// attributeValues translates a product's attribute values. The contract
// already allows only strings and booleans; anything else is reported for
// its attribute.
func attributeValues(in catalogapi.ProductAttributes) (map[string]domain.AttributeValue, error) {
	out := make(map[string]domain.AttributeValue, len(in))
	for key, v := range in {
		switch v := v.(type) {
		case string:
			out[key] = domain.TextValue(v)
		case bool:
			out[key] = domain.BoolValue(v)
		default:
			return nil, &domain.AttributeError{Key: key, Err: domain.ErrInvalidAttributeValue}
		}
	}
	return out, nil
}

// attributesBody builds a product's attribute values for the response.
func attributesBody(a domain.Attributes) catalogapi.ProductAttributes {
	values := a.Values()
	out := make(catalogapi.ProductAttributes, len(values))
	for key, v := range values {
		if v.IsBool() {
			out[key] = v.Bool()
		} else {
			out[key] = v.Text()
		}
	}
	return out
}

// describeAttribute names the attribute of an *AttributeError in the
// message and in details.attribute. format gets the attribute's label, or
// its key when the tenant has no attribute with that key.
func describeAttribute(format func(name string) string) func(error) (string, map[string]string, bool) {
	return func(err error) (string, map[string]string, bool) {
		var attrErr *domain.AttributeError
		if !errors.As(err, &attrErr) {
			return "", nil, false
		}
		name := attrErr.Label
		if name == "" {
			name = attrErr.Key
		}
		return format(name), map[string]string{"attribute": attrErr.Key}, true
	}
}
