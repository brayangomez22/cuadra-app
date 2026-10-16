package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
		c := key[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
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

// resolve checks v against the definition and returns it normalized: text
// trimmed, a number in its canonical form ("2.50" is "2.5") and an option
// spelled as the definition spells it. ok is false for an empty text, which
// counts as no value.
func (d *AttributeDefinition) resolve(v AttributeValue) (_ AttributeValue, ok bool, err error) {
	if d.typ == AttributeBoolean {
		if !v.isBool {
			return AttributeValue{}, false, ErrInvalidAttributeValue
		}
		return v, true, nil
	}
	if v.isBool {
		return AttributeValue{}, false, ErrInvalidAttributeValue
	}
	text := strings.TrimSpace(v.text)
	if text == "" {
		return AttributeValue{}, false, nil
	}
	switch d.typ {
	case AttributeNumber:
		n, err := decimal.NewFromString(text)
		if err != nil || len(text) > maxAttributeNumberLength {
			return AttributeValue{}, false, ErrInvalidAttributeValue
		}
		text = n.String()
	case AttributeList:
		option, found := matchChoice(d.options, text)
		if !found {
			return AttributeValue{}, false, ErrInvalidAttributeValue
		}
		text = option
	default:
		if utf8.RuneCountInString(text) > maxAttributeTextLength {
			return AttributeValue{}, false, ErrInvalidAttributeValue
		}
	}
	return TextValue(text), true, nil
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

// AttributeValue is one attribute's value: a text for text, number and list
// attributes (a number travels as its decimal text, never as a float), or a
// boolean for yes/no ones.
type AttributeValue struct {
	text    string
	boolean bool
	isBool  bool
}

// TextValue is the value of a text, number or list attribute.
func TextValue(s string) AttributeValue { return AttributeValue{text: s} }

// BoolValue is the value of a yes/no attribute.
func BoolValue(b bool) AttributeValue { return AttributeValue{boolean: b, isBool: true} }

// IsBool reports whether the value is a boolean rather than a text.
func (v AttributeValue) IsBool() bool { return v.isBool }

// Text returns the text of a text value; "" for a boolean.
func (v AttributeValue) Text() string { return v.text }

// Bool returns the boolean of a boolean value; false for a text.
func (v AttributeValue) Bool() bool { return v.boolean }

// Attributes are a product's values, keyed by attribute key. Only
// ResolveAttributes builds them from input, so they matched the tenant's
// definitions when they were written. The zero value has no attributes.
type Attributes struct{ values map[string]AttributeValue }

// ResolveAttributes checks in against the tenant's definitions and returns
// the values normalized: every key must be defined, every value must fit
// its type, and every required attribute must have a value (an empty text
// is no value). The error is an *AttributeError naming the attribute.
func ResolveAttributes(defs []*AttributeDefinition, in map[string]AttributeValue) (Attributes, error) {
	byKey := make(map[string]*AttributeDefinition, len(defs))
	for _, d := range defs {
		byKey[d.key] = d
	}
	out := make(map[string]AttributeValue, len(in))
	// Sorted, so the same input always reports the same attribute.
	for _, key := range slices.Sorted(maps.Keys(in)) {
		d, defined := byKey[key]
		if !defined {
			return Attributes{}, &AttributeError{Key: key, Err: ErrUnknownAttribute}
		}
		v, ok, err := d.resolve(in[key])
		if err != nil {
			return Attributes{}, &AttributeError{Key: key, Label: d.label, Err: err}
		}
		if ok {
			out[key] = v
		}
	}
	for _, d := range defs {
		if _, ok := out[d.key]; d.required && !ok {
			return Attributes{}, &AttributeError{Key: d.key, Label: d.label, Err: ErrAttributeRequired}
		}
	}
	return Attributes{values: out}, nil
}

// RehydrateAttributes rebuilds a product's stored values. It does not check
// them against the definitions, which may have changed since they were
// written; it checks only what storage must hold: valid keys and no empty
// texts.
func RehydrateAttributes(values map[string]AttributeValue) (Attributes, error) {
	for key, v := range values {
		if !validAttributeKey(key) || !v.isBool && v.text == "" {
			return Attributes{}, fmt.Errorf("%w: stored attribute %q", ErrInvalidAttributeValue, key)
		}
	}
	return Attributes{values: maps.Clone(values)}, nil
}

// Values returns a copy of the values by key; empty without attributes.
func (a Attributes) Values() map[string]AttributeValue {
	if a.values == nil {
		return map[string]AttributeValue{}
	}
	return maps.Clone(a.values)
}

// AttributeError is an error about one of a product's attributes. Label is
// empty when the tenant has no attribute with Key.
type AttributeError struct {
	Key   string
	Label string
	Err   error
}

func (e *AttributeError) Error() string { return fmt.Sprintf("%v: %q", e.Err, e.Key) }

func (e *AttributeError) Unwrap() error { return e.Err }
