package apispec_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/apispec"
)

const securitySpec = `
openapi: 3.0.3
info: {title: test, version: "1"}
security:
  - bearerAuth: []
paths:
  /healthz:
    get:
      security: []
      responses: {"200": {description: ok}}
  /api/v1/products:
    get:
      responses: {"200": {description: ok}}
    post:
      x-permission: catalog:write
      responses: {"201": {description: ok}}
  /api/v1/products/{id}:
    delete:
      x-permission: catalog:write
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses: {"204": {description: ok}}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
`

func loadSpec(t *testing.T, src string) *openapi3.T {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromData([]byte(src))
	require.NoError(t, err)
	return spec
}

func TestOperationSecurity(t *testing.T) {
	t.Run("lee qué operaciones son públicas y qué permiso exige cada una", func(t *testing.T) {
		got, err := apispec.OperationSecurity(loadSpec(t, securitySpec))
		require.NoError(t, err)

		require.Equal(t, map[string]apispec.Security{
			"GET /healthz":                 {Public: true},
			"GET /api/v1/products":         {},
			"POST /api/v1/products":        {Permission: "catalog:write"},
			"DELETE /api/v1/products/{id}": {Permission: "catalog:write"},
		}, got)
	})

	t.Run("rechaza una operación pública que declara un permiso", func(t *testing.T) {
		spec := loadSpec(t, securitySpec)
		spec.Paths.Find("/healthz").Get.Extensions = map[string]any{"x-permission": "catalog:write"}

		_, err := apispec.OperationSecurity(spec)
		require.Error(t, err)
	})

	t.Run("rechaza un x-permission que no es un texto", func(t *testing.T) {
		spec := loadSpec(t, securitySpec)
		spec.Paths.Find("/api/v1/products").Post.Extensions = map[string]any{"x-permission": 42}

		_, err := apispec.OperationSecurity(spec)
		require.Error(t, err)
	})

	t.Run("en el contrato real las probes y el login son públicos y /me exige token", func(t *testing.T) {
		spec, err := apispec.Load(t.Context())
		require.NoError(t, err)

		got, err := apispec.OperationSecurity(spec)
		require.NoError(t, err)

		for _, op := range []string{
			"GET /healthz", "GET /readyz",
			"POST /api/v1/auth/signup", "POST /api/v1/auth/login",
			"POST /api/v1/auth/refresh", "POST /api/v1/auth/logout",
		} {
			require.True(t, got[op].Public, op)
		}
		require.Equal(t, apispec.Security{}, got["GET /api/v1/me"])
	})
}
