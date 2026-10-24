package usecase

import (
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// Instrument creates one metric instrument of a Service with a meter.
type Instrument func(metric.Meter) error

// Meter returns the meter instrumentationName of mp; a nil mp is a no-op.
func Meter(mp metric.MeterProvider, instrumentationName string) metric.Meter {
	if mp == nil {
		mp = metricnoop.NewMeterProvider()
	}
	return mp.Meter(instrumentationName)
}

// Create creates the instruments in order and stops at the first that fails.
func Create(meter metric.Meter, instruments ...Instrument) error {
	for _, create := range instruments {
		if err := create(meter); err != nil {
			return err
		}
	}
	return nil
}

// Counter creates the counter name into target.
func Counter(target *metric.Int64Counter, name, description, unit string) Instrument {
	return func(meter metric.Meter) error {
		counter, err := meter.Int64Counter(name, metric.WithDescription(description), metric.WithUnit(unit))
		if err != nil {
			return err
		}
		*target = counter
		return nil
	}
}

// Histogram creates the histogram name into target, with the bucket
// boundaries of buckets (the SDK's defaults when there are none).
func Histogram(target *metric.Float64Histogram, name, description, unit string, buckets ...float64) Instrument {
	return func(meter metric.Meter) error {
		options := []metric.Float64HistogramOption{metric.WithDescription(description), metric.WithUnit(unit)}
		if len(buckets) > 0 {
			options = append(options, metric.WithExplicitBucketBoundaries(buckets...))
		}
		histogram, err := meter.Float64Histogram(name, options...)
		if err != nil {
			return err
		}
		*target = histogram
		return nil
	}
}
