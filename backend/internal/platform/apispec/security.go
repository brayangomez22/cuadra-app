package apispec

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// permissionExtension names the permission an operation requires.
const permissionExtension = "x-permission"

// Security is what an operation of the contract demands of the caller.
type Security struct {
	// Public operations declare `security: []`; the rest require a token.
	Public bool
	// Permission comes from the operation's x-permission, if any.
	Permission string
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

			if raw, ok := op.Extensions[permissionExtension]; ok {
				permission, isString := raw.(string)
				if !isString || permission == "" {
					return nil, fmt.Errorf("openapi: %s: %s must be a non-empty string", key, permissionExtension)
				}
				if sec.Public {
					return nil, fmt.Errorf("openapi: %s: a public operation cannot require %q", key, permission)
				}
				sec.Permission = permission
			}
			out[key] = sec
		}
	}
	return out, nil
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
