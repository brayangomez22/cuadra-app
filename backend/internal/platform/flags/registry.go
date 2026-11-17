// Package flags turns features on and off per tenant without a deploy: to
// release something to a pilot first, or to switch off what fails. Each flag
// has a global value and exceptions per tenant, which the platform's
// operators change; a flag nobody turned on is off.
//
// Flags are not modules (CLAUDE.md, rule 10): a module is a capability the
// tenant has (and later pays for), a flag a temporary switch of the team.
// The code reads them through the OpenFeature API (Provider) or, for GET
// /me, with Evaluator.Active.
package flags

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

// keyPattern is the shape of a flag key, the same the tables check.
var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Definition is a flag the code knows.
type Definition struct {
	// Key is the stable ID the code and the frontend ask for.
	Key string
	// Description says what the flag turns on, for the operator screen.
	Description string
}

// Registry is the set of flags the code declares, in declaration order.
type Registry struct {
	defs []Definition
}

// NewRegistry builds a Registry. It fails, so the API does not start, if a
// key is malformed, repeated or has no description.
func NewRegistry(defs ...Definition) (*Registry, error) {
	r := &Registry{}
	for _, d := range defs {
		if !keyPattern.MatchString(d.Key) {
			return nil, fmt.Errorf("flags: key %q is not snake_case", d.Key)
		}
		if d.Description == "" {
			return nil, fmt.Errorf("flags: %s has no description", d.Key)
		}
		if r.Known(d.Key) {
			return nil, fmt.Errorf("flags: %s is declared twice", d.Key)
		}
		r.defs = append(r.defs, d)
	}
	return r, nil
}

// Known reports whether key is a declared flag.
func (r *Registry) Known(key string) bool {
	return slices.ContainsFunc(r.defs, func(d Definition) bool { return d.Key == key })
}

// Definitions returns the declared flags, in declaration order.
func (r *Registry) Definitions() []Definition { return slices.Clone(r.defs) }

// ErrUnknownFlag is the error for a key the code does not declare.
var ErrUnknownFlag = errors.New("flags: unknown flag")
