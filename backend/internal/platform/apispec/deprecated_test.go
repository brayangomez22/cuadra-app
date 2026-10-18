package apispec_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/apispec"
)

// allowedDeprecated lists what the contract may mark as deprecated, as JSON
// paths of the spec ("paths./api/v1/x.get"). The generic core block (T24a–T24j)
// removed everything it deprecated; a field deprecated later, once there are
// clients in production, is added here on purpose.
var allowedDeprecated = []string{}

func TestContractHasNoDeprecated(t *testing.T) {
	t.Run("el contrato no tiene operaciones, parámetros ni campos deprecated", func(t *testing.T) {
		spec, err := apispec.Load(t.Context())
		require.NoError(t, err)
		raw, err := json.Marshal(spec)
		require.NoError(t, err)
		var doc any
		require.NoError(t, json.Unmarshal(raw, &doc))

		var found []string
		walkDeprecated(doc, "", &found)
		found = slices.DeleteFunc(found, func(p string) bool { return slices.Contains(allowedDeprecated, p) })
		require.Empty(t, found)
	})
}

// walkDeprecated appends to found the path of every object of v with
// "deprecated": true. References are not followed: each schema is checked
// where it is defined.
func walkDeprecated(v any, path string, found *[]string) {
	switch v := v.(type) {
	case map[string]any:
		if d, ok := v["deprecated"].(bool); ok && d {
			*found = append(*found, path)
		}
		for k, child := range v {
			walkDeprecated(child, join(path, k), found)
		}
	case []any:
		for i, child := range v {
			walkDeprecated(child, join(path, fmt.Sprint(i)), found)
		}
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
