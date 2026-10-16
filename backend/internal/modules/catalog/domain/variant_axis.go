package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxAxisAllowedValues = 200

// VariantAxis is an axis a tenant's variants vary on (size, color,
// presentation...) and the values it allows. Variants keep their value under
// the axis name, so the name never changes.
type VariantAxis struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	name      string
	values    []string
	createdAt time.Time
}

// NewVariantAxis creates an axis with its own UUID v7. The name is stored
// trimmed and lowercase, as the variants store it.
func NewVariantAxis(tenantID uuid.UUID, name string, values []string, now time.Time) (*VariantAxis, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newVariantAxis(VariantAxisSnapshot{ID: id, TenantID: tenantID, Name: name, Values: values, CreatedAt: now})
}

// newVariantAxis checks the invariants shared by NewVariantAxis and
// RehydrateVariantAxis.
func newVariantAxis(s VariantAxisSnapshot) (*VariantAxis, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidAxisID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	name, ok := normalizeAxisName(s.Name)
	if !ok {
		return nil, ErrInvalidAxisName
	}
	a := &VariantAxis{id: s.ID, tenantID: s.TenantID, name: name, createdAt: s.CreatedAt.UTC()}
	if _, err := a.Update(s.Values); err != nil {
		return nil, err
	}
	return a, nil
}

// normalizeAxisName trims and lowercases name, as a variant's axis values
// name their axes.
func normalizeAxisName(name string) (string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	return name, name != "" && utf8.RuneCountInString(name) <= maxAxisNameLength
}

// ID returns the axis ID (UUID v7).
func (a *VariantAxis) ID() uuid.UUID { return a.id }

// TenantID returns the tenant the axis belongs to.
func (a *VariantAxis) TenantID() uuid.UUID { return a.tenantID }

// Name returns the axis name, lowercase ("color").
func (a *VariantAxis) Name() string { return a.name }

// Values returns a copy of the values the axis allows, in their order.
func (a *VariantAxis) Values() []string { return slices.Clone(a.values) }

// CreatedAt returns the creation time in UTC.
func (a *VariantAxis) CreatedAt() time.Time { return a.createdAt }

// Update replaces the allowed values and returns the ones it removed: the
// repository refuses the change if a variant has one of them. An axis
// allows at least one value, and no two that differ only in case. On error
// the axis is unchanged.
func (a *VariantAxis) Update(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrInvalidAxisAllowedValues
	}
	normalized, err := normalizeChoices(values, maxAxisAllowedValues, maxAxisValueLength, ErrInvalidAxisAllowedValues)
	if err != nil {
		return nil, err
	}
	removed := removedChoices(a.values, normalized)
	a.values = normalized
	return removed, nil
}

// VariantAxisSnapshot holds a stored axis's fields, for RehydrateVariantAxis.
type VariantAxisSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Values    []string
	CreatedAt time.Time
}

// RehydrateVariantAxis rebuilds a stored axis, checking the same invariants
// as NewVariantAxis.
func RehydrateVariantAxis(s VariantAxisSnapshot) (*VariantAxis, error) { return newVariantAxis(s) }

// ResolveAxisValues checks a variant's axis values against the tenant's
// axes and returns them with each name normalized and each value spelled as
// its axis spells it, so "m" is the size "M". The error for an undefined
// axis or value is an *AxisError naming the axis.
func ResolveAxisValues(axes []*VariantAxis, in map[string]string) (map[string]string, error) {
	byName := make(map[string]*VariantAxis, len(axes))
	for _, a := range axes {
		byName[a.name] = a
	}
	out := make(map[string]string, len(in))
	// Sorted, so the same input always reports the same axis.
	for _, raw := range slices.Sorted(maps.Keys(in)) {
		name, ok := normalizeAxisName(raw)
		if !ok {
			return nil, ErrInvalidAxisValues
		}
		if _, dup := out[name]; dup {
			return nil, ErrInvalidAxisValues
		}
		a, defined := byName[name]
		if !defined {
			return nil, &AxisError{Axis: name, Err: ErrUnknownAxis}
		}
		value, allowed := matchChoice(a.values, strings.TrimSpace(in[raw]))
		if !allowed {
			return nil, &AxisError{Axis: name, Err: ErrAxisValueNotAllowed}
		}
		out[name] = value
	}
	return out, nil
}

// AxisError is an error about one of a variant's axes.
type AxisError struct {
	Axis string
	Err  error
}

func (e *AxisError) Error() string { return fmt.Sprintf("%v: %q", e.Err, e.Axis) }

func (e *AxisError) Unwrap() error { return e.Err }
