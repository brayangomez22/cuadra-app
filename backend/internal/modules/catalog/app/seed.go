package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Seed is a new tenant's initial catalog configuration, as its business
// template declares it: units of measure, root categories, attribute
// definitions and variant axes.
type Seed struct {
	Units       []UnitSeed
	Categories  []string
	Attributes  []domain.AttributeParams
	VariantAxes []AxisSeed
}

// UnitSeed is a unit of measure of a Seed.
type UnitSeed struct{ Code, Name string }

// AxisSeed is a variant axis of a Seed and the values it allows.
type AxisSeed struct {
	Name   string
	Values []string
}

// SeedTenant gives a new tenant its initial catalog, all or nothing. Every
// entity is built with its constructor, so a seed with an invalid unit,
// category, attribute or axis fails before anything is stored. It runs at
// sign-up, before the tenant has a user acting on it.
func (s *Service) SeedTenant(ctx context.Context, tenantID uuid.UUID, in Seed) (err error) {
	ctx, span := s.spans.Start(ctx, "SeedTenant", attribute.String("tenant.id", tenantID.String()))
	defer func() { usecase.End(span, err) }()

	seed, err := s.buildSeed(tenantID, in)
	if err != nil {
		return err
	}
	return s.Seeds.Seed(ctx, tenantID, seed)
}

func (s *Service) buildSeed(tenantID uuid.UUID, in Seed) (domain.TenantSeed, error) {
	now := s.Now()
	var seed domain.TenantSeed
	for _, u := range in.Units {
		unit, err := domain.NewUnit(tenantID, u.Code, u.Name, now)
		if err != nil {
			return domain.TenantSeed{}, err
		}
		seed.Units = append(seed.Units, unit)
	}
	for _, name := range in.Categories {
		c, err := domain.NewCategory(tenantID, name, nil, now)
		if err != nil {
			return domain.TenantSeed{}, err
		}
		seed.Categories = append(seed.Categories, c)
	}
	for _, p := range in.Attributes {
		d, err := domain.NewAttributeDefinition(tenantID, p, now)
		if err != nil {
			return domain.TenantSeed{}, err
		}
		seed.Attributes = append(seed.Attributes, d)
	}
	for _, a := range in.VariantAxes {
		axis, err := domain.NewVariantAxis(tenantID, a.Name, a.Values, now)
		if err != nil {
			return domain.TenantSeed{}, err
		}
		seed.VariantAxes = append(seed.VariantAxes, axis)
	}
	return seed, nil
}
