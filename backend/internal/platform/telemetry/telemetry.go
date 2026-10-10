// Package telemetry sets up the OpenTelemetry SDK: traces, metrics and logs,
// exported over OTLP/HTTP. Exporters are configured with the standard OTEL_*
// environment variables (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_RESOURCE_ATTRIBUTES...).
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Resource identifies the service in every signal.
type Resource struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
}

// ShutdownFunc flushes pending telemetry and stops the providers.
type ShutdownFunc func(context.Context) error

// Setup installs the global tracer, meter and logger providers and the W3C
// propagator. When OTEL_SDK_DISABLED=true it installs nothing: the global
// no-op providers stay in place and the app runs without a collector.
func Setup(ctx context.Context, res Resource, getenv func(string) string) (ShutdownFunc, error) {
	if strings.EqualFold(strings.TrimSpace(getenv("OTEL_SDK_DISABLED")), "true") {
		return func(context.Context) error { return nil }, nil
	}

	r, err := newResource(ctx, res)
	if err != nil {
		return nil, err
	}

	var shutdowns []ShutdownFunc
	shutdown := func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			errs = append(errs, fn(ctx))
		}
		return errors.Join(errs...)
	}
	fail := func(err error) (ShutdownFunc, error) {
		return nil, errors.Join(err, shutdown(ctx))
	}

	traceExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return fail(fmt.Errorf("trace exporter: %w", err))
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExp), sdktrace.WithResource(r))
	shutdowns = append(shutdowns, tp.Shutdown)

	metricExp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return fail(fmt.Errorf("metric exporter: %w", err))
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)), sdkmetric.WithResource(r))
	shutdowns = append(shutdowns, mp.Shutdown)

	logExp, err := otlploghttp.New(ctx)
	if err != nil {
		return fail(fmt.Errorf("log exporter: %w", err))
	}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)), sdklog.WithResource(r))
	shutdowns = append(shutdowns, lp.Shutdown)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return shutdown, nil
}

// newResource describes the service. OTEL_SERVICE_NAME and
// OTEL_RESOURCE_ATTRIBUTES override the values given in code.
func newResource(ctx context.Context, res Resource) (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceName(res.ServiceName),
			semconv.ServiceVersion(res.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(res.Environment),
		),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	}
	r, err := resource.New(ctx, attrs...)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}
	return r, nil
}
