package apispec

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// permissionExtension names the permission an operation requires, and
// moduleExtension the activable module it belongs to.
const (
	permissionExtension = "x-permission"
	moduleExtension     = "x-module"
)

// Security is what an operation of the contract demands of the caller.
type Security struct {
	// Public operations declare `security: []`; the rest require a token.
	Public bool
	// Permission comes from the operation's x-permission, if any.
	Permission string
	// Module comes from the operation's x-module, if any.
	Module string
}

// OperationSecurity returns the security of every operation in spec, keyed
// like the generated routes ("POST /api/v1/products/{id}"). An operation
// without its own `security` inherits the document's; one is public when its
// requirements are empty or include an empty one. It fails if a public
// operation declares a permission or x-permission is not a string.
func OperationSecurity(spec *openapi3.T) (map[string]Security, error) {
	out := map[string]Security{}
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			key := strings.ToUpper(method) + " " + path

			requirements := spec.Security
			if op.Security != nil {
				requirements = *op.Security
			}
			sec := Security{Public: isPublic(requirements)}

			var err error
			if sec.Permission, err = extension(op, key, permissionExtension, sec.Public); err != nil {
				return nil, err
			}
			if sec.Module, err = extension(op, key, moduleExtension, sec.Public); err != nil {
				return nil, err
			}
			out[key] = sec
		}
	}
	return out, nil
}

// extension returns the string extension name of op, "" when it has none.
func extension(op *openapi3.Operation, key, name string, public bool) (string, error) {
	raw, ok := op.Extensions[name]
	if !ok {
		return "", nil
	}
	value, isString := raw.(string)
	if !isString || value == "" {
		return "", fmt.Errorf("openapi: %s: %s must be a non-empty string", key, name)
	}
	if public {
		return "", fmt.Errorf("openapi: %s: a public operation cannot declare %s %q", key, name, value)
	}
	return value, nil
}

func isPublic(requirements openapi3.SecurityRequirements) bool {
	if len(requirements) == 0 {
		return true
	}
	for _, r := range requirements {
		if len(r) == 0 {
			return true
		}
	}
	return false
}
