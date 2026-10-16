package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// AttributeInput is the data of a new attribute definition.
type AttributeInput struct {
	Key      string
	Label    string
	Type     domain.AttributeType
	Options  []string
	Required bool
}

// AttributeUpdate is the editable data of a definition: its key and type
// never change, because products keep their values under the key.
type AttributeUpdate struct {
	Label    string
	Options  []string
	Required bool
}

// CreateAttribute stores a new attribute definition.
func (s *Service) CreateAttribute(ctx context.Context, a Actor, in AttributeInput) (_ *domain.AttributeDefinition, err error) {
	ctx, span := s.startSpan(ctx, "CreateAttribute", a)
	defer func() { endSpan(span, err) }()

	d, err := domain.NewAttributeDefinition(a.TenantID, domain.AttributeParams{
		Key: in.Key, Label: in.Label, Type: in.Type, Options: in.Options, Required: in.Required,
	}, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attributeAttr(d.ID()))
	if err := s.Attributes.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateAttribute changes a definition's label, options and whether it is
// required. An option a product uses cannot be removed. A product without a
// newly required attribute keeps its values until it is edited.
func (s *Service) UpdateAttribute(ctx context.Context, a Actor, id uuid.UUID, in AttributeUpdate) (_ *domain.AttributeDefinition, err error) {
	ctx, span := s.startSpan(ctx, "UpdateAttribute", a, attributeAttr(id))
	defer func() { endSpan(span, err) }()

	d, err := s.Attributes.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	removed, err := d.Update(in.Label, in.Options, in.Required)
	if err != nil {
		return nil, err
	}
	if err := s.Attributes.Update(ctx, d, removed); err != nil {
		return nil, err
	}
	return d, nil
}

// ListAttributes returns every attribute definition of the tenant, by label.
func (s *Service) ListAttributes(ctx context.Context, a Actor) (_ []*domain.AttributeDefinition, err error) {
	ctx, span := s.startSpan(ctx, "ListAttributes", a)
	defer func() { endSpan(span, err) }()

	return s.Attributes.List(ctx, a.TenantID)
}

// DeleteAttribute removes a definition no product has a value for.
func (s *Service) DeleteAttribute(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteAttribute", a, attributeAttr(id))
	defer func() { endSpan(span, err) }()

	return s.Attributes.Delete(ctx, a.TenantID, id)
}

// resolveAttributes checks a product's attribute values against the
// tenant's definitions.
func (s *Service) resolveAttributes(ctx context.Context, a Actor, in map[string]domain.AttributeValue) (domain.Attributes, error) {
	defs, err := s.Attributes.List(ctx, a.TenantID)
	if err != nil {
		return domain.Attributes{}, err
	}
	return domain.ResolveAttributes(defs, in)
}
