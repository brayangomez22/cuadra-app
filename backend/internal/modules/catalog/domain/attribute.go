package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxAttributeKeyLength    = 50
	maxAttributeLabelLength  = 100
	maxAttributeOptions      = 100
	maxAttributeOptionLength = 100
	maxAttributeTextLength   = 500
	maxAttributeNumberLength = 40
)

// AttributeType is the kind of value an attribute holds. The types are the
// same for every business; what each tenant defines is its attributes.
type AttributeType string

// The attribute types: free text, a decimal number, one option of a list,
// and yes/no.
const (
	AttributeText    AttributeType = "text"
	AttributeNumber  AttributeType = "number"
	AttributeList    AttributeType = "list"
	AttributeBoolean AttributeType = "boolean"
)

// ParseAttributeType returns the type named s; ErrInvalidAttributeType if
// there is none.
func ParseAttributeType(s string) (AttributeType, error) {
	switch t := AttributeType(s); t {
	case AttributeText, AttributeNumber, AttributeList, AttributeBoolean:
		return t, nil
	}
	return "", ErrInvalidAttributeType
}

// AttributeDefinition is a product attribute a tenant defines (brand,
// material, active ingredient...). Products keep their values under its
// key, so the key and the type never change.
type AttributeDefinition struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	key       string
	label     string
	typ       AttributeType
	options   []string
	required  bool
	createdAt time.Time
}

// AttributeParams holds the data of a new attribute definition. Options are
// for list attributes only.
type AttributeParams struct {
	Key      string
	Label    string
	Type     AttributeType
	Options  []string
	Required bool
}

// NewAttributeDefinition creates a definition with its own UUID v7.
func NewAttributeDefinition(tenantID uuid.UUID, p AttributeParams, now time.Time) (*AttributeDefinition, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newAttributeDefinition(AttributeSnapshot{
		ID: id, TenantID: tenantID, Key: p.Key, Label: p.Label, Type: p.Type,
		Options: p.Options, Required: p.Required, CreatedAt: now,
	})
}

// newAttributeDefinition checks the invariants shared by
// NewAttributeDefinition and RehydrateAttributeDefinition.
func newAttributeDefinition(s AttributeSnapshot) (*AttributeDefinition, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidAttributeID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if !validAttributeKey(s.Key) {
		return nil, ErrInvalidAttributeKey
	}
	if _, err := ParseAttributeType(string(s.Type)); err != nil {
		return nil, err
	}
	d := &AttributeDefinition{id: s.ID, tenantID: s.TenantID, key: s.Key, typ: s.Type, createdAt: s.CreatedAt.UTC()}
	if _, err := d.Update(s.Label, s.Options, s.Required); err != nil {
		return nil, err
	}
	return d, nil
}

// validAttributeKey reports whether key is a lowercase ASCII slug that
// starts with a letter: it is stable for templates, imports and filters.
func validAttributeKey(key string) bool {
	if key == "" || len(key) > maxAttributeKeyLength || key[0] < 'a' || key[0] > 'z' {
		return false
	}
	for i := 0; i < len(key); i++ {
		if !isKeyChar(key[i]) {
			return false
		}
	}
	return true
}

// isKeyChar reports whether c may appear in an attribute key: [a-z0-9_].
func isKeyChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

// normalizeChoices trims each choice of a list of options or axis values,
// keeping their order. None may be empty nor longer than maxLength, and no
// two may differ only in case: a value is matched case-insensitively.
func normalizeChoices(in []string, maxCount, maxLength int, errInvalid error) ([]string, error) {
	if len(in) > maxCount {
		return nil, errInvalid
	}
	out := make([]string, 0, len(in))
	for _, c := range in {
		c = strings.TrimSpace(c)
		if c == "" || utf8.RuneCountInString(c) > maxLength {
			return nil, errInvalid
		}
		if _, ok := matchChoice(out, c); ok {
			return nil, errInvalid
		}
		out = append(out, c)
	}
	return out, nil
}

// matchChoice returns the choice equal to value without regard to case.
func matchChoice(choices []string, value string) (string, bool) {
	i := slices.IndexFunc(choices, func(c string) bool { return strings.EqualFold(c, value) })
	if i < 0 {
		return "", false
	}
	return choices[i], true
}

// removedChoices returns the choices of before that after no longer has,
// spelled exactly: changing an option's case removes the old spelling.
func removedChoices(before, after []string) []string {
	var removed []string
	for _, c := range before {
		if !slices.Contains(after, c) {
			removed = append(removed, c)
		}
	}
	return removed
}

// ID returns the definition ID (UUID v7).
func (d *AttributeDefinition) ID() uuid.UUID { return d.id }

// TenantID returns the tenant the definition belongs to.
func (d *AttributeDefinition) TenantID() uuid.UUID { return d.tenantID }

// Key returns the key products keep the value under ("marca").
func (d *AttributeDefinition) Key() string { return d.key }

// Label returns the name shown to the user ("Marca").
func (d *AttributeDefinition) Label() string { return d.label }

// Type returns the kind of value the attribute holds.
func (d *AttributeDefinition) Type() AttributeType { return d.typ }

// Options returns a copy of a list attribute's options, in their order; nil
// for the other types.
func (d *AttributeDefinition) Options() []string { return slices.Clone(d.options) }

// Required reports whether every product must have a value.
func (d *AttributeDefinition) Required() bool { return d.required }

// CreatedAt returns the creation time in UTC.
func (d *AttributeDefinition) CreatedAt() time.Time { return d.createdAt }

// Update replaces the label, the options and whether the attribute is
// required, and returns the options it removed: the repository refuses the
// change if a product has one of them. A list needs at least one option and
// the other types take none. On error the definition is unchanged.
func (d *AttributeDefinition) Update(label string, options []string, required bool) ([]string, error) {
	label, err := normalizeText(label, maxAttributeLabelLength, ErrInvalidAttributeLabel)
	if err != nil {
		return nil, err
	}
	if (d.typ == AttributeList) != (len(options) > 0) {
		return nil, ErrInvalidAttributeOptions
	}
	var normalized []string
	if d.typ == AttributeList {
		if normalized, err = normalizeChoices(options, maxAttributeOptions, maxAttributeOptionLength, ErrInvalidAttributeOptions); err != nil {
			return nil, err
		}
	}
	removed := removedChoices(d.options, normalized)
	d.label, d.options, d.required = label, normalized, required
	return removed, nil
}

// AttributeSnapshot holds a stored definition's fields, for
// RehydrateAttributeDefinition.
type AttributeSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Key       string
	Label     string
	Type      AttributeType
	Options   []string
	Required  bool
	CreatedAt time.Time
}

// RehydrateAttributeDefinition rebuilds a stored definition, checking the
// same invariants as NewAttributeDefinition.
func RehydrateAttributeDefinition(s AttributeSnapshot) (*AttributeDefinition, error) {
	return newAttributeDefinition(s)
}
