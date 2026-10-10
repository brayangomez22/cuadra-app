package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetup(t *testing.T) {
	t.Run("con OTEL_SDK_DISABLED=true no instala proveedores y el shutdown no falla", func(t *testing.T) {
		getenv := func(key string) string {
			if key == "OTEL_SDK_DISABLED" {
				return "true"
			}
			return ""
		}

		shutdown, err := Setup(context.Background(), Resource{ServiceName: "cuadra-api"}, getenv)

		require.NoError(t, err)
		require.NotNil(t, shutdown)
		_, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider)
		require.False(t, isSDK, "no debe instalar el TracerProvider del SDK")
		require.NoError(t, shutdown(context.Background()))
	})
}

func TestNewResource(t *testing.T) {
	t.Run("el recurso incluye nombre de servicio, versión y ambiente", func(t *testing.T) {
		res, err := newResource(context.Background(), Resource{
			ServiceName:    "cuadra-api",
			ServiceVersion: "abc123",
			Environment:    "development",
		})

		require.NoError(t, err)
		attrs := map[attribute.Key]string{}
		for _, kv := range res.Attributes() {
			attrs[kv.Key] = kv.Value.String()
		}
		require.Equal(t, "cuadra-api", attrs["service.name"])
		require.Equal(t, "abc123", attrs["service.version"])
		require.Equal(t, "development", attrs["deployment.environment.name"])
	})
}
