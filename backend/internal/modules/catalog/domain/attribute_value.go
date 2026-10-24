package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

// resolve checks v against the definition and returns it normalized: text
// trimmed, a number in its canonical form ("2.50" is "2.5") and an option
// spelled as the definition spells it. ok is false for an empty text, which
// counts as no value.
func (d *AttributeDefinition) resolve(v AttributeValue) (_ AttributeValue, ok bool, err error) {
	if (d.typ == AttributeBoolean) != v.isBool {
		return AttributeValue{}, false, ErrInvalidAttributeValue
	}
	if v.isBool {
		return v, true, nil
	}
	text := strings.TrimSpace(v.text)
	if text == "" {
		return AttributeValue{}, false, nil
	}
	text, err = d.resolveText(text)
	if err != nil {
		return AttributeValue{}, false, err
	}
	return TextValue(text), true, nil
}

// resolveText checks a non-empty text against a text, number or list
// definition and returns it normalized.
func (d *AttributeDefinition) resolveText(text string) (string, error) {
	switch d.typ {
	case AttributeNumber:
		n, err := decimal.NewFromString(text)
		if err != nil || len(text) > maxAttributeNumberLength {
			return "", ErrInvalidAttributeValue
		}
		return n.String(), nil
	case AttributeList:
		option, found := matchChoice(d.options, text)
		if !found {
			return "", ErrInvalidAttributeValue
		}
		return option, nil
	default:
		if utf8.RuneCountInString(text) > maxAttributeTextLength {
			return "", ErrInvalidAttributeValue
		}
		return text, nil
	}
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
