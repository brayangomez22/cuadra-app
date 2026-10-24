// Package modules is the mechanism of Cuadra's activable modules: the
// registry of the modules that exist, with their dependencies and
// permissions, and the service that turns them on and off for each tenant.
//
// The core never imports a module. A module's operations declare it in the
// contract (x-module), and the API answers 404 when the tenant does not have
// it active; the frontend shows menus, columns and fields by the modules
// GET /api/v1/me lists.
package modules

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Errors of the registry (at startup) and of the service.
var (
	ErrDuplicateModule    = errors.New("modules: module registered twice")
	ErrUnknownDependency  = errors.New("modules: dependency is not registered")
	ErrDependencyCycle    = errors.New("modules: dependency cycle")
	ErrInvalidModuleID    = errors.New("modules: invalid module id")
	ErrInvalidModuleName  = errors.New("modules: invalid module name")
	ErrUnknownModule      = errors.New("modules: unknown module")
	ErrDependencyInactive = errors.New("modules: a dependency of the module is inactive")
	ErrRequiredBy         = errors.New("modules: an active module depends on this one")
)

// DependencyError names the module that blocks a change: the inactive
// dependency (ErrDependencyInactive) or the active module that depends on
// the one being turned off (ErrRequiredBy).
type DependencyError struct {
	Err error
	// Module is the blocking module's ID, and Name its name.
	Module string
	Name   string
}

func (e *DependencyError) Error() string { return fmt.Sprintf("%v: %s", e.Err, e.Module) }
func (e *DependencyError) Unwrap() error { return e.Err }

// Permission is a permission a module declares, "<namespace>:<action>",
// and the roles other than the owner that get it. The identity module
// validates both when it builds the permission matrix.
type Permission struct {
	ID    string
	Roles []string
}

// Definition declares a module.
type Definition struct {
	// ID is stable: it is stored in tenant_modules and named by the
	// contract's x-module. Lowercase letters, digits and underscores.
	ID string
	// Name is shown to the user.
	Name string
	// DependsOn are the modules that must be active for this one to be.
	DependsOn   []string
	Permissions []Permission
}

func (d Definition) clone() Definition {
	d.DependsOn = slices.Clone(d.DependsOn)
	perms := make([]Permission, len(d.Permissions))
	for i, p := range d.Permissions {
		perms[i] = Permission{ID: p.ID, Roles: slices.Clone(p.Roles)}
	}
	d.Permissions = perms
	return d
}

var moduleIDFormat = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Registry is the set of modules that exist. It is built once, at startup,
// and never changes.
type Registry struct {
	defs  []Definition
	index map[string]int
}

// NewRegistry registers defs in their order, which is the order the modules
// are listed in. It fails, so the API does not start, if an ID is malformed
// or repeated, a module has no name, or a dependency is unknown or cyclic.
func NewRegistry(defs ...Definition) (*Registry, error) {
	r := &Registry{defs: make([]Definition, 0, len(defs)), index: make(map[string]int, len(defs))}
	for _, d := range defs {
		if err := r.add(d); err != nil {
			return nil, err
		}
	}
	if err := r.checkDependencies(); err != nil {
		return nil, err
	}
	return r, nil
}

// add registers d after checking its ID and name.
func (r *Registry) add(d Definition) error {
	if !moduleIDFormat.MatchString(d.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidModuleID, d.ID)
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("%w: %s", ErrInvalidModuleName, d.ID)
	}
	if _, dup := r.index[d.ID]; dup {
		return fmt.Errorf("%w: %s", ErrDuplicateModule, d.ID)
	}
	r.index[d.ID] = len(r.defs)
	r.defs = append(r.defs, d.clone())
	return nil
}

// checkDependencies checks that every dependency is registered and that
// none is cyclic.
func (r *Registry) checkDependencies() error {
	for _, d := range r.defs {
		for _, dep := range d.DependsOn {
			if _, ok := r.index[dep]; !ok {
				return fmt.Errorf("%w: %s depends on %s", ErrUnknownDependency, d.ID, dep)
			}
		}
	}
	for _, d := range r.defs {
		if err := r.checkCycle(d.ID, nil); err != nil {
			return err
		}
	}
	return nil
}

// checkCycle walks id's dependencies depth first; path holds the modules
// on the way to id.
func (r *Registry) checkCycle(id string, path []string) error {
	path = append(slices.Clip(path), id)
	if slices.Contains(path[:len(path)-1], id) {
		return fmt.Errorf("%w: %s", ErrDependencyCycle, strings.Join(path, " → "))
	}
	for _, dep := range r.defs[r.index[id]].DependsOn {
		if err := r.checkCycle(dep, path); err != nil {
			return err
		}
	}
	return nil
}

// Definitions returns a copy of every module, in registration order.
func (r *Registry) Definitions() []Definition {
	out := make([]Definition, len(r.defs))
	for i, d := range r.defs {
		out[i] = d.clone()
	}
	return out
}

// Known reports whether id is a registered module.
func (r *Registry) Known(id string) bool {
	_, ok := r.index[id]
	return ok
}

// get returns the module id; ok is false when it is not registered.
func (r *Registry) get(id string) (Definition, bool) {
	i, ok := r.index[id]
	if !ok {
		return Definition{}, false
	}
	return r.defs[i], true
}

// CheckSet checks that the modules ids can be turned on together, as a
// business template does for a new tenant: each one is registered and
// listed once, and its dependencies are among ids. Otherwise it fails with
// ErrUnknownModule, ErrDuplicateModule or a *DependencyError
// (ErrDependencyInactive).
func (r *Registry) CheckSet(ids []string) error {
	for i, id := range ids {
		if !r.Known(id) {
			return fmt.Errorf("%w: %s", ErrUnknownModule, id)
		}
		if slices.Contains(ids[:i], id) {
			return fmt.Errorf("%w: %s", ErrDuplicateModule, id)
		}
	}
	for _, id := range ids {
		def, _ := r.get(id)
		for _, dep := range def.DependsOn {
			if !slices.Contains(ids, dep) {
				d, _ := r.get(dep)
				return &DependencyError{Err: ErrDependencyInactive, Module: dep, Name: d.Name}
			}
		}
	}
	return nil
}
